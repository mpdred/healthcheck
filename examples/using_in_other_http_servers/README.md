```go
package main

import (
	"context"
	"log"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

	log.Println("create probes ...")
	deadmansProbe := factories.NewProbeBuilder().BuildDeadmansSnitch()

	log.Println("register probes ...")
	probeStore.Add(deadmansProbe)

	// Now let's assume that you have an echoserver (https://github.com/labstack/echo) running,
	// and you wish echoserver to handle the probe we just created.

	// Echo instance
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.GET("/", hello)

	// Now we will add the routes of the Healthcheck
	for _, endpointDefinition := range endpointDefinitions {
		def := endpointDefinition

		// Define a handle func that is compatible with echo server
		fn := func(c echo.Context) error {
			def.HandleFunc(c.Response(), c.Request())

			return nil
		}

		e.Any(endpointDefinition.Endpoint, fn)
	}

	e.Logger.Fatal(e.Start(":1323"))

	// You can now check the /live, /ready, /health, or /metrics endpoint.
	// E.g.:

	// $ curl localhost:1323/live -v
	// *   Trying 127.0.0.1:1323...
	// * Connected to localhost (127.0.0.1) port 1323 (#0)
	// > GET /live HTTP/1.1
	// > Host: localhost:1323
	// > User-Agent: curl/7.85.0
	// > Accept: */*
	// >
	// * Mark bundle as not supporting multiuse
	// < HTTP/1.1 204 No Content
	// < Date: Sun, 15 Jan 2023 17:27:51 GMT
	// <
	// * Connection #0 to host localhost left intact

	// Now if we check the metrics on the healthcheck server (mux),
	// we can see that the probe execution has been recorded.

	// $ curl localhost:5059/metrics -v | grep health
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
	// < Date: Sun, 15 Jan 2023 17:28:48 GMT
	// < Transfer-Encoding: chunked
	// <
	// { [3956 bytes data]
	// 100  5038    0  5038    0     0   682k      0 --:--:-- --:--:-- --:--:-- 4919k
	// * Connection #0 to host localhost left intact
	// # HELP my_namespace_healthcheck_status Current probe check status (0=healthy, 1=unhealthy)
	// # TYPE my_namespace_healthcheck_status gauge
	// my_namespace_healthcheck_status{kind="liveness",probe="dead man's snitch"} 0
}

// Handler
func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

```
