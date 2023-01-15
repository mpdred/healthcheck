package healthcheck

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type EndpointDefinition struct {
	Name       string
	Endpoint   string
	HandleFunc func(h http.ResponseWriter, r *http.Request)
}

// GetHandleFuncForEchoServer returns a handlerfunc that is compatible with echo server (https://github.com/labstack/echo).
func (d EndpointDefinition) GetHandleFuncForEchoServer() echo.HandlerFunc {
	fn := func(c echo.Context) error {
		d.HandleFunc(c.Response(), c.Request())

		return nil
	}

	return fn

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
