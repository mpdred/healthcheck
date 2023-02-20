package factories

import (
	"encoding/json"
	"fmt"
	"log"
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

			var isProbeCheckFailed bool
			errMessages := map[string]string{}
			for _, executionResult := range executionResults {
				if executionResult.Probe.Health == healthcheck.UnhealthyStatus {
					isProbeCheckFailed = true
					errMessages[executionResult.Probe.Name] = executionResult.Err.Error()
				}
			}

			if !isProbeCheckFailed {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			w.WriteHeader(http.StatusServiceUnavailable)

			jsonStr, err := json.Marshal(errMessages)
			if err != nil {
				log.Printf("Error: %s\n", err.Error())
			}

			_, err = w.Write(jsonStr)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println(err)
			}
		}

		probeFns[k] = fn
	}

	return probeFns
}
