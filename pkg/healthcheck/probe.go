package healthcheck

import (
	"context"
)

type ProbeCheckFn func(context.Context) error

// ProbeKind is a superset of Kubernetes probe kinds:
//
//   - LivenessProbeKind: The kubelet uses liveness probes to know when to restart a container. For example, liveness probes could catch a deadlock, where an application is running, but unable to make progress. Restarting a container in such a state can help to make the application more available despite bugs.
//   - ReadinessProbeKind: The kubelet uses readiness probes to know when a container is ready to start accepting traffic. A Pod is considered ready when all of its containers are ready. One use of this signal is to control which Pods are used as backends for Services. When a Pod is not ready, it is removed from Service load balancers.
//   - StartupProbeKind: The kubelet uses startup probes to know when a container application has started. If such a probe is configured, it disables liveness and readiness checks until it succeeds, making sure those probes don't interfere with the application startup. This can be used to adopt liveness checks on slow starting containers, avoiding them getting killed by the kubelet before they are up and running.
//
// Source: https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/
//
// An additional CustomProbeKind kind is provided, which should be triggered on-demand.
type ProbeKind string

const (
	LivenessProbeKind  ProbeKind = "liveness"
	ReadinessProbeKind ProbeKind = "readiness"
	StartupProbeKind   ProbeKind = "startup"

	CustomProbeKind ProbeKind = "custom"
)

// ProbeHealthStatus is the status of a Probe's CheckF
type ProbeHealthStatus string

const (
	HealthyStatus   ProbeHealthStatus = "healthy"
	UnhealthyStatus ProbeHealthStatus = "unhealthy"
)

type Probe struct {
	CheckFn ProbeCheckFn
	Kind    ProbeKind
	Name    string
	Health  ProbeHealthStatus
}

func (p Probe) Execute(ctx context.Context) error {
	err := p.CheckFn(ctx)

	return err
}
