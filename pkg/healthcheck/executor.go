package healthcheck

import (
	"context"
	"sync"
)

type Executor interface {
	Execute(ctx context.Context, probes []Probe) []ExecutionResult
}

type ExecutionResult struct {
	Probe Probe  `json:"probe"`
	Err   string `json:"err,omitempty"`
}

type executor struct{}

func (e executor) Execute(ctx context.Context, probes []Probe) []ExecutionResult {
	var wg sync.WaitGroup
	c := make(chan ExecutionResult)

	wg.Add(len(probes))

	go func() {
		wg.Wait()
		close(c)
	}()

	for _, p := range probes {
		go func(p Probe) {
			defer wg.Done()
			r := ExecutionResult{
				Probe: p,
			}

			err := p.Execute(ctx)
			if err != nil {
				r.Err = err.Error()
			}

			c <- r
		}(p)
	}

	rr := make([]ExecutionResult, 0, len(probes))
	for r := range c {
		rr = append(rr, r)
	}

	return rr
}

func NewExecutor() Executor {
	var e Executor = &executor{}

	return e
}
