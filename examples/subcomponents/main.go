package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/mpdred/healthcheck/v2/pkg/factories"
	"github.com/mpdred/healthcheck/v2/pkg/healthcheck"
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

	// You can make a map of your components and use it to decide if the app is ready or not.
	componentsStatus := map[string]bool{
		"foo": true,
		"bar": false,
		"baz": true,
	}

	probes := factories.NewProbeBuilder().BuildForComponents(healthcheck.ReadinessProbeKind, componentsStatus)

	log.Println("register probes ...")
	probeStore.Add(probes...)

	// Here we are simulating that the component 'foo' changes its status due to outside conditions,
	// And we're expecting that the Prometheus metric will change accordingly.
	go func() {
		rand.Seed(time.Now().UTC().UnixNano())
		for {
			componentsStatus["foo"] = rand.Intn(2) == 1
			log.Printf("changing the status of component 'foo' to %t\n", componentsStatus["foo"])
			time.Sleep(1 * time.Second)
		}
	}()

	log.Println("keeping the http server open for you ...")
	fmt.Println("Press <Enter> to exit...")
	input := bufio.NewScanner(os.Stdin)
	input.Scan()
	log.Println("main() finished")

	// You can now check the /live, /ready, /health, or /metrics endpoint.
	// E.g.:

	// $ curl -v localhost:5059/health
	// *   Trying 127.0.0.1:5059...
	// * Connected to localhost (127.0.0.1) port 5059 (#0)
	// > GET /health HTTP/1.1
	// > Host: localhost:5059
	// > User-Agent: curl/7.85.0
	// > Accept: */*
	// >
	// * Mark bundle as not supporting multiuse
	// < HTTP/1.1 503 Service Unavailable
	// < Date: Sun, 15 Jan 2023 16:14:39 GMT
	// < Content-Length: 0
	// <
	// * Connection #0 to host localhost left intact

	// On watching the metrics in a loop you can see that
	// the value of the metric (the last line in the following example) changes:

	// Every 0.3s: curl --silent localhost:5059/health && curl -v localhost:5059/metrics | grep health                                                                                            Mariuss-MacBook-Pro.local: Sun Jan 15 18:34:[0/690]
	//
	// *   Trying 127.0.0.1:5059...
	//  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
	//                                 Dload  Upload   Total   Spent    Left  Speed
	//   0     0    0     0    0     0      0      0 --:--:-- --:--:-- --:--:--     0* Connected to localhost (127.0.0.1) port 5059 (#0)
	// > GET /metrics HTTP/1.1
	// > Host: localhost:5059
	// > User-Agent: curl/7.85.0
	// > Accept: */*
	// >
	// * Mark bundle as not supporting multiuse
	// < HTTP/1.1 200 OK
	// < Content-Type: text/plain; version=0.0.4; charset=utf-8
	// < Date: Sun, 15 Jan 2023 16:34:10 GMT
	// < Transfer-Encoding: chunked
	// <
	// { [5239 bytes data]
	// 100  5213    0  5213    0     0   680k      0 --:--:-- --:--:-- --:--:-- 5090k
	// * Connection #0 to host localhost left intact
	// # HELP my_namespace_healthcheck_status Current probe check status (0=healthy, 1=unhealthy)
	// # TYPE my_namespace_healthcheck_status gauge
	// my_namespace_healthcheck_status{kind="readiness",probe="component bar"} 1
	// my_namespace_healthcheck_status{kind="readiness",probe="component baz"} 0
	// my_namespace_healthcheck_status{kind="readiness",probe="component foo"} 1

	// The value from 'my_namespace_healthcheck_status{kind="readiness",probe="component foo"} 1' will change from 0 to 1 as defined in the go func above.
}
