package healthcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Handler interface {
	http.Handler
	Worker

	RegisterProbes(probes ...Probe)
	GetProbes(kind ProbeKind) []Probe
	Execute(ctx context.Context, kind ProbeKind) []ExecutionResult
}

type handler struct {
	endpoints map[string]string
	executor  Executor
	probes    map[string]Probe

	http.ServeMux

	server          *http.Server
	serverCtx       context.Context
	serverCancelCtx context.CancelFunc

	prometheusStatusGauge *prometheus.GaugeVec
}

func (h *handler) RegisterProbes(probes ...Probe) {
	for _, p := range probes {
		h.probes[p.GetName()] = p
	}
}

func (h *handler) handleEndpoints(w http.ResponseWriter, r *http.Request) {
	err := json.NewEncoder(w).Encode(h.endpoints)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *handler) GetProbes(kind ProbeKind) []Probe {
	pp := make([]Probe, 0)
	for _, p := range h.probes {
		if p.GetKind() == kind {
			pp = append(pp, p)
		}
	}

	return pp
}

func (h *handler) handleLiveness(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, Liveness)
}

func (h *handler) handleReadiness(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, Readiness)
}

func (h *handler) handle(w http.ResponseWriter, r *http.Request, kind ProbeKind) {
	rr := h.Execute(r.Context(), kind)

	for _, r := range rr {
		if r.Err != nil {
			h.prometheusStatusGauge.WithLabelValues(r.Probe.GetName()).Set(1)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		h.prometheusStatusGauge.WithLabelValues(r.Probe.GetName()).Set(0)
	}

	w.WriteHeader(204)
}

func (h *handler) Execute(ctx context.Context, kind ProbeKind) []ExecutionResult {
	probesToExecute := h.GetProbes(kind)
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
	}, []string{"probe"})
}

func NewHandler(port int, executor Executor, namespace string, registry prometheus.Registerer) Handler {
	const (
		MetricsName     = "metrics"
		MetricsEndpoint = "/" + MetricsName
	)

	h := &handler{
		executor: executor,
		endpoints: map[string]string{
			string(Liveness):  "/live",
			string(Readiness): "/ready",
			MetricsName:       MetricsEndpoint,
		},
		probes: map[string]Probe{},
	}

	registry.MustRegister()

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.handleEndpoints)
	mux.HandleFunc(h.endpoints[string(Liveness)], h.handleLiveness)
	mux.HandleFunc(h.endpoints[string(Readiness)], h.handleReadiness)

	mux.Handle(MetricsEndpoint, promhttp.Handler())
	h.initGauges(namespace)

	ctx, cancelCtx := context.WithCancel(context.Background())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			ctx = context.WithValue(ctx, "serverAddr", l.Addr().String())
			return ctx
		},
	}

	h.server = server
	h.serverCtx = ctx
	h.serverCancelCtx = cancelCtx

	return h
}
