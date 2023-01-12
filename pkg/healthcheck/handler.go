package healthcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handler interface {
	http.Handler
	Worker

	RegisterProbes(probes ...*Probe)
	GetProbe(name string) *Probe
	UnregisterProbes(names ...string)

	// GetProbesByKind returns all probes that have a matching ProbeKind.
	//
	// ProbeKind = Health returns all probes.
	GetProbesByKind(kind ProbeKind) []*Probe

	// Execute all the probes of this ProbeKind.
	//
	// This can be triggered on demand or when a http server isn't started.
	Execute(ctx context.Context, kind ProbeKind) []ExecutionResult

	// HandleHealth executes all the Probe(s)
	// and updates the Prometheus Gauge(0=success, 1=failure).
	// Probes that have completed their ProbeCheckFn successfully won't return an error.
	//
	// Returns http.StatusOK and the list of ExecutionResult(s).
	HandleHealth(w http.ResponseWriter, r *http.Request)

	// HandleLiveness executes all the Probe(s) of ProbeKind = Liveness
	// and updates the Prometheus Gauge (0=success, 1=failure).
	//
	// Returns http.StatusOK if all the ProbeCheckFn(s) have completed successfully,
	// or http.StatusServiceUnavailable when at least one probe has failed.
	HandleLiveness(w http.ResponseWriter, r *http.Request)

	// HandleReadiness executes all the Probe(s) of ProbeKind = Readiness
	// and updates the Prometheus Gauge (0=success, 1=failure).
	//
	// Returns http.StatusOK if all the ProbeCheckFn(s) have completed successfully,
	// or http.StatusServiceUnavailable when at least one probe has failed.
	HandleReadiness(w http.ResponseWriter, r *http.Request)

	// HandleStartup executes all the Probe(s) of ProbeKind = Startup
	// and updates the Prometheus Gauge (0=success, 1=failure).
	//
	// Returns http.StatusOK if all the ProbeCheckFn(s) have completed successfully,
	// or http.StatusServiceUnavailable when at least one probe has failed.
	HandleStartup(w http.ResponseWriter, r *http.Request)
}

type handler struct {
	mu sync.Mutex

	endpoints map[string]string
	executor  Executor
	probes    map[string]*Probe

	http.ServeMux

	server          *http.Server
	serverCtx       context.Context
	serverCancelCtx context.CancelFunc

	prometheusStatusGauge *prometheus.GaugeVec
}

func (h *handler) GetProbe(name string) *Probe {
	p, ok := h.probes[name]
	if !ok {
		return nil
	}

	return p
}

func (h *handler) RegisterProbes(probes ...*Probe) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, p := range probes {
		h.probes[p.Name] = p
	}
}

func (h *handler) UnregisterProbes(names ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, name := range names {
		delete(h.probes, name)
	}
}

func (h *handler) handleEndpoints(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(h.endpoints)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *handler) GetProbesByKind(kind ProbeKind) []*Probe {
	pp := make([]*Probe, 0)
	for _, p := range h.probes {
		if kind == Health {
			pp = append(pp, p)
			continue
		}

		if p.Kind == kind {
			pp = append(pp, p)
		}
	}

	return pp
}

func (h *handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	resList := h.Execute(r.Context(), Health)
	h.updatePrometheusGauge(resList)

	err := json.NewEncoder(w).Encode(resList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *handler) HandleLiveness(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, Liveness)
}

func (h *handler) HandleReadiness(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, Readiness)
}

func (h *handler) HandleStartup(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, Startup)
}

func (h *handler) handle(w http.ResponseWriter, r *http.Request, kind ProbeKind) {
	resList := h.Execute(r.Context(), kind)
	h.updatePrometheusGauge(resList)

	var hasAtLeastOneErr bool
	for _, res := range resList {
		p := res.Probe
		if res.Err != "" {
			if !p.IsInformationalOnly {
				hasAtLeastOneErr = true
			}
		}
	}

	if hasAtLeastOneErr {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(200)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *handler) updatePrometheusGauge(resList []ExecutionResult) {
	for _, res := range resList {
		p := res.Probe
		if res.Err != "" {
			h.prometheusStatusGauge.WithLabelValues(string(p.Kind), p.Name).Set(1)
			continue
		}

		h.prometheusStatusGauge.WithLabelValues(string(p.Kind), p.Name).Set(0)
	}
}

func (h *handler) Execute(ctx context.Context, kind ProbeKind) []ExecutionResult {
	probesToExecute := h.GetProbesByKind(kind)
	return h.executor.Execute(ctx, probesToExecute)
}

func (h *handler) Start() {
	go func() {
		err := h.server.ListenAndServe()
		if err != nil {
			fmt.Printf("error listening for health check server: %s\n", err)
		}
		h.serverCancelCtx()
	}()

	<-h.serverCtx.Done()
}

func (h *handler) Stop() {
	err := h.server.Close()
	if err != nil {
		fmt.Printf("error closing health check server: %s\n", err)
	}

	h.serverCancelCtx()
	h.serverCtx.Done()
}

func (h *handler) initGauges(namespace string) {
	h.prometheusStatusGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "healthcheck",
		Name:      "status",
		Help:      "Current check status (0=success, 1=failure)",
	}, []string{"kind", "probe"})
}

func NewHandler(port int, executor Executor, namespace string, registry prometheus.Registerer) Handler {
	const (
		HealthName      = "health"
		HealthEndpoint  = "/" + HealthName
		MetricsName     = "metrics"
		MetricsEndpoint = "/" + MetricsName
		StartupName     = "startup"
		StartupEndpoint = "/" + StartupName
	)

	h := &handler{
		executor: executor,
		endpoints: map[string]string{
			string(Health):    HealthEndpoint,
			string(Liveness):  "/live",
			MetricsName:       MetricsEndpoint,
			string(Readiness): "/ready",
			string(Startup):   StartupEndpoint,
		},
		probes: map[string]*Probe{},
	}

	registry.MustRegister()

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleEndpoints)
	mux.HandleFunc(h.endpoints[string(Health)], h.HandleHealth)
	mux.HandleFunc(h.endpoints[string(Liveness)], h.HandleLiveness)
	mux.HandleFunc(h.endpoints[string(Readiness)], h.HandleReadiness)
	mux.HandleFunc(h.endpoints[string(Startup)], h.HandleStartup)

	mux.Handle(MetricsEndpoint, promhttp.Handler())
	h.initGauges(namespace)

	ctx, cancelCtx := context.WithCancel(context.Background())

	type serverAddr struct{}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			ctx = context.WithValue(ctx, serverAddr{}, l.Addr().String())
			return ctx
		},
	}

	h.server = server
	h.serverCtx = ctx
	h.serverCancelCtx = cancelCtx

	return h
}
