package factories

import (
	"fmt"
	"net/http"

	"github.com/mpdred/healthcheck/v2/pkg/healthcheck"
)

func NewMuxHandler(endpoints []healthcheck.EndpointDefinition, metricsService healthcheck.MetricsService) *http.ServeMux {
	mux := http.NewServeMux()
	for _, endpoint := range endpoints {
		mux.HandleFunc(endpoint.Endpoint, endpoint.HandleFunc)
	}

	mux.Handle(healthcheck.MetricsEndpoint, metricsService.GetHandler())

	return mux
}

func GetEndpointDefinitions(service healthcheck.Service) []healthcheck.EndpointDefinition {
	probeExecutionFns := getProbeExecutionFns(service)

	endpoints := make([]healthcheck.EndpointDefinition, 0, len(probeExecutionFns))

	for endpointKind := range healthcheck.EndpointDefinitions {
		for probeExecutionFnKind, probeExecutionFn := range probeExecutionFns {
			if endpointKind != probeExecutionFnKind {
				continue
			}

			endpoint := healthcheck.EndpointDefinitions[endpointKind]
			endpoint.HandleFunc = probeExecutionFn

			healthcheck.EndpointDefinitions[endpointKind] = endpoint
			endpoints = append(endpoints, endpoint)
		}
	}

	return endpoints
}

func getProbeExecutionFns(service healthcheck.Service) map[healthcheck.ProbeKind]func(w http.ResponseWriter, r *http.Request) {
	probeFns := map[healthcheck.ProbeKind]func(w http.ResponseWriter, r *http.Request){}
	for _, kind := range []healthcheck.ProbeKind{healthcheck.StartupProbeKind, healthcheck.LivenessProbeKind, healthcheck.ReadinessProbeKind, healthcheck.CustomProbeKind} {
		k := kind
		fn := func(w http.ResponseWriter, r *http.Request) {
			var executionResults []healthcheck.ExecutionResult
			var err error

			if k == healthcheck.CustomProbeKind {
				executionResults, err = service.ExecuteAllProbes(r.Context())
			} else {
				executionResults, err = service.ExecuteProbesByKind(r.Context(), k)
			}

			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, err := w.Write([]byte(err.Error()))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Println(err)
				}
			}

			for _, executionResult := range executionResults {
				if executionResult.Probe.Health == healthcheck.UnhealthyStatus {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
			}

			w.WriteHeader(http.StatusNoContent)
		}

		probeFns[k] = fn
	}

	return probeFns
}
