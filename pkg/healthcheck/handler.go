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

	RegisterProbes(probes ...Probe)
	GetProbe(name string) *Probe
	UnregisterProbes(names ...string)

	// GetProbesByKind returns all probes that have a matching ProbeKind.
	//
	// ProbeKind = Health returns all probes.
	GetProbesByKind(kind ProbeKind) []Probe
	Execute(ctx context.Context, kind ProbeKind) []ExecutionResult
}

type handler struct {
	mu sync.Mutex

	endpoints map[string]string
	executor  Executor
	probes    map[string]Probe

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

	return &p
}

func (h *handler) RegisterProbes(probes ...Probe) {
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

func (h *handler) GetProbesByKind(kind ProbeKind) []Probe {
	pp := make([]Probe, 0)
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

func (h *handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	rr := h.executionResults(w, r, Health)

	err := json.NewEncoder(w).Encode(rr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *handler) handleLiveness(w http.ResponseWriter, r *http.Request) {
	h.executionResults(w, r, Liveness)
}

func (h *handler) handleReadiness(w http.ResponseWriter, r *http.Request) {
	h.executionResults(w, r, Readiness)
}

func (h *handler) handleStartup(w http.ResponseWriter, r *http.Request) {
	h.executionResults(w, r, Startup)
}

func (h *handler) executionResults(w http.ResponseWriter, r *http.Request, kind ProbeKind) []ExecutionResult {
	rr := h.Execute(r.Context(), kind)

	var hasAtLeastOneErr bool
	for _, r := range rr {
		p := r.Probe
		if r.Err != "" {
			h.prometheusStatusGauge.WithLabelValues(string(p.Kind), p.Name).Set(1)

			if !p.IsInformationalOnly {
				hasAtLeastOneErr = true
				continue
			}
		}

		h.prometheusStatusGauge.WithLabelValues(string(p.Kind), p.Name).Set(0)
	}

	if hasAtLeastOneErr {
		w.WriteHeader(http.StatusServiceUnavailable)
		return nil
	}

	if kind == Health {
		return rr
	}

	w.WriteHeader(200)
	_, err := w.Write([]byte("OK"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	return rr
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
		probes: map[string]Probe{},
	}

	registry.MustRegister()

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleEndpoints)
	mux.HandleFunc(h.endpoints[string(Health)], h.handleHealth)
	mux.HandleFunc(h.endpoints[string(Liveness)], h.handleLiveness)
	mux.HandleFunc(h.endpoints[string(Readiness)], h.handleReadiness)
	mux.HandleFunc(h.endpoints[string(Startup)], h.handleStartup)

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
