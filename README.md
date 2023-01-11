# [healthcheck](https://github.com/mpdred/healthcheck)

## Summary

Healthcheck is a library for implementing liveness and readiness checks to your Go app, with Prometheus metrics support.

## Endpoints

| name       | description                                                            |
|------------|------------------------------------------------------------------------|
| `/metrics` | Publishes Prometheus metrics                                           |
| `/live`    | Used by Kubernetes probes to see if the app is running                 |
| `/ready`   | Used by Kubernetes probes to see if the app is ready to accept traffic |

## How to Use

See the [examples](examples/main.go).
