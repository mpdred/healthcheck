package healthcheck

import (
	"context"
	"errors"
)

var ErrCheckFailed = errors.New("probe check failed")

type ProbeCheckFn func(context.Context) error

type ProbeKind string

const (
	Health    ProbeKind = "health"
	Liveness  ProbeKind = "liveness"
	Readiness ProbeKind = "readiness"
	Startup   ProbeKind = "startup"
)

type Probe struct {
	checkFn ProbeCheckFn
	Kind    ProbeKind `json:"kind"`
	Name    string    `json:"name"`
}

func (p Probe) Execute(ctx context.Context) error {
	err := p.checkFn(ctx)

	return err
}
