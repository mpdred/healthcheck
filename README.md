# healthcheck ![go version](https://img.shields.io/github/go-mod/go-version/mpdred/healthcheck) ![GitHub Actions](https://img.shields.io/github/actions/workflow/status/mpdred/healthcheck/go.yaml) ![tag](https://img.shields.io/github/v/tag/mpdred/healthcheck) ![last commit](https://img.shields.io/github/last-commit/mpdred/healthcheck)

[(source)](https://github.com/mpdred/healthcheck)

## Summary

Healthcheck is a library for implementing liveness and readiness checks to your Go app, with Prometheus metrics support.
You can also add your custom checks.

In addition, you can choose to set one or more of the healthcheck endpoints in your app's http server, if you have one. This is recommended especially for the `live` endpoint.

## Installation

```shell
// go get github.com/mpdred/healthcheck/{version}
go get github.com/mpdred/healthcheck/v2
```

## Quickstart

```golang
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
```

The default http server will serve requests, and the metrics will be updated as soon as the endpoints are called (calling `/metrics` doesn't refresh the metrics since it doesn't do any checks by itself).

You can see this example [here](./examples/sandbox.go)

## Endpoints and ports

The default port is `5090`.

| endpoint   | response code                                  | description                                                                                                                                                      |
|------------|------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `/health`  | 204 No Content <br/>OR 503 Service Unavailable | Informational health check statuses, that shouldn't be taken into account by any Kubernetes probe. It executes all the probes defined, regardless of their kind. |
| `/live`    | 204 No Content <br/>OR 503 Service Unavailable | Can be used by the Kubernetes liveness probe to see if the app is running.                                                                                       |
| `/metrics` | 200 OK                                         | Publishes Prometheus metrics. Note: The metrics are generated and/or updated only when the other endpoints are called.                                           |
| `/ready`   | 204 No Content <br/>OR 503 Service Unavailable | Can be used by the Kubernetes readiness probe to see if the app is ready to accept traffic.                                                                      |
| `/startup` | 204 No Content <br/>OR 503 Service Unavailable | Can be used by the Kubernetes startup probe to see if the app has been initialized successfully.                                                                 |

## Metrics

Currently only Prometheus metrics are supported, but feel free to open a pull request if you want to add more!

If you don't need metrics you can ignore them by using the provided `noopMetricsService`.

### Prometheus

A Gauge is created with the user-provided namespace, in the subsystem `healthcheck`, with the name `status`. It has two labels: `kind` and `probe`.

The gauge value is the status: 0=healthy, 1=unhealthy.
E.g.:

- if the probe check is successful:

> my_namespace_healthcheck_status{kind="liveness",probe="dead man's snitch"} 0

- if the probe check failed

> my_namespace_healthcheck_status{kind="liveness",probe="dead man's snitch"} 1

## Probes

Probes are the building block of this library, and some predefined checks for probes have been defined in [ProbeBuilder](./pkg/factories/ProbeBuilder). This includes HTTP GET, DNS resolve, and TCP dial calls, and SQL, Redis, and Opensearch connectivity checks.

The probe checks are done async,

## Other examples

See the [examples](./examples/README.md).
