package healthcheck

import (
	"net/http"
)

type EndpointDefinition struct {
	Name       string
	Endpoint   string
	HandleFunc func(h http.ResponseWriter, r *http.Request)
}

const (
	StartupName     = "startup"
	StartupEndpoint = "/startup"

	LivenessName     = "liveness"
	LivenessEndpoint = "/live"

	ReadinessName     = "readiness"
	ReadinessEndpoint = "/ready"

	HealthName     = "health"
	HealthEndpoint = "/health"

	MetricsName     = "metrics"
	MetricsEndpoint = "/metrics"
)

var (
	EndpointDefinitions = map[ProbeKind]EndpointDefinition{
		StartupName: {
			Name:     StartupName,
			Endpoint: StartupEndpoint,
		},
		LivenessName: {
			Name:     LivenessName,
			Endpoint: LivenessEndpoint,
		},
		ReadinessName: {
			Name:     ReadinessName,
			Endpoint: ReadinessEndpoint,
		},
		HealthName: {
			Name:     HealthName,
			Endpoint: HealthEndpoint,
		},
	}
)
