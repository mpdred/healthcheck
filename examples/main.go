package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mpdred/healthcheck/v2/pkg/healthcheck"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	// Create some probes
	probeFnFactory := healthcheck.NewProbeFnFactory()

	dialCheckFn := probeFnFactory.TCPDialWithTimeout("google.com:443", 5*time.Second)
	dialProbe := healthcheck.NewProbe("dial google", dialCheckFn, healthcheck.Liveness)

	httpCheckFn := probeFnFactory.HTTPGetCheck("https://www.google.com", 5*time.Second)
	httpProbe := healthcheck.NewProbe("get google", httpCheckFn, healthcheck.Readiness)

	customProbe := healthcheck.NewProbe(
		"bar baz",
		func(ctx context.Context) error {
			return errors.New("an unexpected error has occurred")
		},
		healthcheck.Readiness)

	// Create server
	e := healthcheck.NewExecutor()
	h := healthcheck.NewHandler(9091, e, "mynamespace", prometheus.NewRegistry())
	h.RegisterProbes(*dialProbe, *httpProbe, *customProbe)

	fmt.Println("start healthcheck server...")
	go h.Start()
	defer h.Stop()
	time.Sleep(10 * time.Second)
	fmt.Println("doing other things")
	fmt.Println("stopping main()")
}
