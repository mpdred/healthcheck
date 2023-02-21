package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mpdred/healthcheck/v2/pkg/factories"
	"github.com/mpdred/healthcheck/v2/pkg/healthcheck"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	ctx := context.Background()

	log.Println("initialize the http server and dependencies ...")
	probeStore := healthcheck.NewInMemoryProbeStore()

	reg := prometheus.NewRegistry()
	opts := promhttp.HandlerOpts{
		ErrorLog: log.Default(),
		Registry: reg,
	}

	promHandler := promhttp.HandlerFor(reg, opts)

	metricsService := healthcheck.NewPrometheusMetricsServiceWithHandler("my_namespace", reg, promHandler)
	service := healthcheck.NewService(probeStore, metricsService)

	endpointDefinitions := factories.GetEndpointDefinitions(service)
	handler := factories.NewMuxHandler(endpointDefinitions, metricsService)
	httpServer := factories.NewServerBuilder().WithPort(5059).WithHandler(handler).Build(ctx)

	log.Println("http server started")
	go healthcheck.StartHTTPServer(httpServer)
	defer healthcheck.StopHTTPServer(httpServer)
	log.Println("http server started")

	log.Println("create probes ...")
	deadmansProbe := factories.NewProbeBuilder().BuildDeadmansSnitch()

	log.Println("register probes ...")
	probeStore.Add(deadmansProbe)

	log.Println("keeping the http server open for you ...")
	fmt.Println("Press <Enter> to exit...")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	log.Println("main() finished")
}
