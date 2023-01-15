package main

import (
	"context"

	"github.com/mpdred/healthcheck/v2/pkg/factories"
	"github.com/mpdred/healthcheck/v2/pkg/healthcheck"
	"github.com/pkg/errors"
)

func main() {
	ctx := context.Background()
	probeStore := healthcheck.NewInMemoryProbeStore()
	metricsService := healthcheck.NewPrometheusMetricsService("my_namespace")
	service := healthcheck.NewService(probeStore, metricsService)
	endpointDefinitions := factories.GetEndpointDefinitions(service)
	handler := factories.NewMuxHandler(endpointDefinitions, metricsService)
	httpServer := factories.NewServerBuilder().WithPort(5059).WithHandler(handler).Build(ctx)

	go healthcheck.StartHTTPServer(httpServer)
	defer healthcheck.StopHTTPServer(httpServer)

	deadmansProbe := factories.NewProbeBuilder().BuildDeadmansSnitch()

	customProbe := factories.NewProbeBuilder().
		WithName("my custom probe").
		WithKind(healthcheck.CustomProbeKind).
		WithCustomCheck(func(context.Context) error {
			return errors.New("an unexpected error has occurred")
		}).
		MustBuild()

	probeStore.Add(deadmansProbe, customProbe)
}
