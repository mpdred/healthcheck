package main

import (
	"context"
	"log"
	"time"

	"github.com/mpdred/healthcheck/v2/pkg/healthcheck"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	log.Println("create health check server...")
	e := healthcheck.NewExecutor()
	h := healthcheck.NewHandler(9999, e, "mynamespace", prometheus.NewRegistry())

	log.Println("start healthcheck server...")
	go h.Start()
	defer h.Stop()

	// Create some probes
	deadmansProbe := healthcheck.NewProbeBuilder().BuildDeadmansSnitch()

	dialCheckProbe := healthcheck.NewProbeBuilder().
		WithTCPDialWithTimeoutCheck("google.com:443").
		WithKind(healthcheck.Startup).
		Build()

	httpCheckProbe := healthcheck.NewProbeBuilder().
		WithHTTPGetCheck("https://www.google.com").
		WithKind(healthcheck.Readiness).
		Build()

	customProbe := healthcheck.NewProbeBuilder().
		WithName("my custom probe").
		WithKind(healthcheck.Health).
		WithCustomCheck(func(context.Context) error {
			return errors.New("an unexpected error has occurred")
		}).
		MustBuild()

	// Register the probes
	h.RegisterProbes(*deadmansProbe, *dialCheckProbe, *httpCheckProbe, *customProbe)

	log.Println("doing other things...")
	time.Sleep(5 * time.Minute)
	log.Println("main() finished")

	// $ curl localhost:9999/health
	//
	// $ curl localhost:9999/metrics | grep health
	//
	//   % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
	//                                 Dload  Upload   Total   Spent    Left  Speed
	// 100  5162    0  5162    0     0  1138k      0 --:--:-- --:--:-- --:--:-- 5041k
	// # HELP mynamespace_healthcheck_status Current check status (0=success, 1=failure)
	// # TYPE mynamespace_healthcheck_status gauge
	// mynamespace_healthcheck_status{probe="dead man's snitch"} 0
	// mynamespace_healthcheck_status{probe="http get"} 0
	// mynamespace_healthcheck_status{probe="my custom probe"} 1
	// mynamespace_healthcheck_status{probe="tcp dial"} 0
}
