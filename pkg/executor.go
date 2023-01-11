package pkg

import (
	"context"
)

type Executor interface {
	Execute(ctx context.Context, probes []Probe) []ExecutionResult
}

type ExecutionResult struct {
	Probe Probe
	Err   error
}

type executor struct{}

func (e executor) Execute(ctx context.Context, probes []Probe) []ExecutionResult {
	rr := make([]ExecutionResult, 0)

	for _, p := range probes {
		err := p.Execute(ctx)
		r := ExecutionResult{
			Probe: p,
			Err:   err,
		}

		rr = append(rr, r)
	}

	return rr
}

func NewExecutor() Executor {
	var e Executor = &executor{}

	return e
}
