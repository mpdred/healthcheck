package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mpdred/healthcheck/v2/pkg/factories"
	"github.com/mpdred/healthcheck/v2/pkg/healthcheck"
	"github.com/pkg/errors"
)

func main() {
	ctx := context.Background()

	log.Println("initialize the http server and dependencies ...")
	probeStore := healthcheck.NewInMemoryProbeStore()
	metricsService := healthcheck.NewPrometheusMetricsService("my_namespace")
	service := healthcheck.NewService(probeStore, metricsService)

	endpointDefinitions := factories.GetEndpointDefinitions(service)
	handler := factories.NewMuxHandler(endpointDefinitions, metricsService)
	httpServer := factories.NewServerBuilder().WithPort(5059).WithHandler(handler).Build(ctx)

	go healthcheck.StartHTTPServer(httpServer)
	defer healthcheck.StopHTTPServer(httpServer)
	log.Println("http server started")

	log.Println("create probes ...")
	deadmansProbe := factories.NewProbeBuilder().BuildDeadmansSnitch()

	dialCheckProbe := factories.NewProbeBuilder().
		WithTCPDialWithTimeoutCheck("google.com:443").
		WithKind(healthcheck.StartupProbeKind).
		Build()

	httpCheckProbe := factories.NewProbeBuilder().
		WithHTTPGetCheck("https://www.google.com").
		WithKind(healthcheck.ReadinessProbeKind).
		Build()

	customProbe := factories.NewProbeBuilder().
		WithName("my custom probe").
		WithKind(healthcheck.CustomProbeKind).
		WithCustomCheck(func(context.Context) error {
			return errors.New("an unexpected error has occurred")
		}).
		MustBuild()

	log.Println("register probes ...")
	probeStore.Add(deadmansProbe, dialCheckProbe, httpCheckProbe, customProbe)

	log.Println("keeping the http server open for you ...")
	fmt.Println("Press <Enter> to exit...")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	log.Println("main() finished")

	// You can now check the /live, /ready, /health, or /metrics endpoint.
	// E.g.:

	// $ curl -v localhost:5059/live
	// *   Trying 127.0.0.1:5059...
	// * Connected to localhost (127.0.0.1) port 5059 (#0)
	// > GET /live HTTP/1.1
	// > Host: localhost:5059
	// > User-Agent: curl/7.85.0
	// > Accept: */*
	// >
	// * Mark bundle as not supporting multiuse
	// < HTTP/1.1 204 No Content
	// < Date: Sun, 15 Jan 2023 16:04:58 GMT
	// <
	// * Connection #0 to host localhost left intact

	// $ curl -v localhost:5059/metrics | grep health
	// *   Trying 127.0.0.1:5059...
	//  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
	//                                 Dload  Upload   Total   Spent    Left  Speed
	//  0     0    0     0    0     0      0      0 --:--:-- --:--:-- --:--:--     0* Connected to localhost (127.0.0.1) port 5059 (#0)
	// > GET /metrics HTTP/1.1
	// > Host: localhost:5059
	// > User-Agent: curl/7.85.0
	// > Accept: */*
	// >
	// * Mark bundle as not supporting multiuse
	// < HTTP/1.1 200 OK
	// < Content-Type: text/plain; version=0.0.4; charset=utf-8
	// < Date: Sun, 15 Jan 2023 16:06:02 GMT
	// < Transfer-Encoding: chunked
	// <
	// { [5468 bytes data]
	// 100  5442    0  5442    0     0   745k      0 --:--:-- --:--:-- --:--:-- 5314k
	// * Connection #0 to host localhost left intact
	// # HELP my_namespace_healthcheck_status Current probe check status (0=healthy, 1=degraded, 2=unhealthy)
	// # TYPE my_namespace_healthcheck_status gauge
	// my_namespace_healthcheck_status{error="",kind="liveness",probe="dead man's snitch"} 0
	// my_namespace_healthcheck_status{error="",kind="readiness",probe="http get"} 0
	// my_namespace_healthcheck_status{error="",kind="startup",probe="tcp dial"} 0
	// my_namespace_healthcheck_status{error="an unexpected error has occurred",kind="health",probe="my custom probe"} 1
}
