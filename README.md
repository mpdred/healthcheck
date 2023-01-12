# [healthcheck](https://github.com/mpdred/healthcheck)

## Summary

Healthcheck is a library for implementing liveness and readiness checks to your Go app, with Prometheus metrics support.

## Endpoints

| name       | description                                                                                        |
|------------|----------------------------------------------------------------------------------------------------|
| `/health`  | Informational health check statuses, that shouldn't be taken into account by any Kubernetes probe. |
| `/live`    | Used by Kubernetes liveness probe to see if the app is running.                                    |
| `/metrics` | Publishes Prometheus metrics.                                                                      |
| `/ready`   | Used by Kubernetes readiness probe to see if the app is ready to accept traffic.                   |
| `/startup` | Used by Kubernetes startup probe to see if the app is running.                                     |

## How to Use

See the [examples](examples/main.go).
