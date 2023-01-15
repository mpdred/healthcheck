package healthcheck

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsService interface {
	UpdateGauge(executionResults ...ExecutionResult)
	GetHandler() http.Handler
}

type noopMetricsService struct{}

func (s noopMetricsService) GetHandler() http.Handler {
	return http.NewServeMux()
}

func (s noopMetricsService) UpdateGauge(...ExecutionResult) {
	return
}

func NewNoOpMetricsService() MetricsService {
	s := &noopMetricsService{}

	return s
}

type prometheusMetricsService struct {
	statusGauge *prometheus.GaugeVec
}

func (s prometheusMetricsService) GetHandler() http.Handler {
	return promhttp.Handler()
}

func (s prometheusMetricsService) UpdateGauge(executionResults ...ExecutionResult) {
	wg := sync.WaitGroup{}

	for _, executionResult := range executionResults {
		wg.Add(1)

		go func(e ExecutionResult) {
			defer wg.Done()

			p := e.Probe
			switch p.Health {
			case HealthyStatus:
				s.statusGauge.WithLabelValues(string(p.Kind), p.Name).Set(0)
			case UnhealthyStatus:
				s.statusGauge.WithLabelValues(string(p.Kind), p.Name).Set(1)
			}

		}(executionResult)
	}

	wg.Wait()
}

func NewPrometheusMetricsService(namespace string) MetricsService {
	statusGauge := promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: "healthcheck",
		Name:      "status",
		Help:      fmt.Sprintf("Current probe check status (0=%s, 1=%s)", HealthyStatus, UnhealthyStatus),
	}, []string{"kind", "probe"})

	s := &prometheusMetricsService{
		statusGauge: statusGauge,
	}

	return s
}
