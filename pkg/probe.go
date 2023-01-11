package pkg

import (
	"context"
	"errors"
)

var ErrCheckFailed = errors.New("probe check failed")

type ProbeCheckFn func(context.Context) error

type ProbeKind string

const (
	Readiness ProbeKind = "readiness"
	Liveness  ProbeKind = "liveness"
)

type Probe struct {
	checkFn ProbeCheckFn
	kind    ProbeKind
	name    string
}

func (p Probe) GetKind() ProbeKind {
	return p.kind
}

func (p Probe) GetName() string {
	return p.name
}

func (p Probe) Execute(ctx context.Context) error {
	err := p.checkFn(ctx)

	return err
}

func NewProbe(name string, fn ProbeCheckFn, kind ProbeKind) *Probe {
	c := &Probe{
		name:    name,
		checkFn: fn,
		kind:    kind,
	}

	return c
}
