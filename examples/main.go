package main

import (
	"context"
	"time"

	"github.com/mpdred/healthcheck/v2/pkg"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	// Create some probes
	probeFnFactory := pkg.NewProbeFnFactory()

	dialCheckFn := probeFnFactory.TCPDialWithTimeout("google.com:443", 5*time.Second)
	dialProbe := pkg.NewProbe("dial google", dialCheckFn, pkg.Liveness)

	httpCheckFn := probeFnFactory.HTTPGetCheck("https://www.google.com", 5*time.Second)
	httpProbe := pkg.NewProbe("get google", httpCheckFn, pkg.Readiness)

	customProbe := pkg.NewProbe(
		"bar baz",
		func(ctx context.Context) error {
			return errors.New("an unexpected error has occurred")
		},
		pkg.Readiness)

	// Create server
	e := pkg.NewExecutor()
	h := pkg.NewHandler(9091, e, "mynamespace", prometheus.NewRegistry())
	h.RegisterProbes(*dialProbe, *httpProbe, *customProbe)

	go h.Start()

	time.Sleep(10 * time.Minute)
}
