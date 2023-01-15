package healthcheck

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

var ErrCheckFailed = errors.New("probe check failed")

type Service interface {
	ExecuteAllProbes(ctx context.Context) ([]ExecutionResult, error)

	// ExecuteProbes executes the ProbeCheckFn of the Probe(s).
	ExecuteProbes(ctx context.Context, probes ...Probe) ([]ExecutionResult, error)

	// ExecuteProbesByKind uses ExecuteProbes on all the probes of this ProbeKind.
	ExecuteProbesByKind(ctx context.Context, kind ProbeKind) ([]ExecutionResult, error)
}

type service struct {
	metricsService MetricsService
	probeStore     ProbeStore
}

func (s service) ExecuteAllProbes(ctx context.Context) ([]ExecutionResult, error) {
	probes := s.probeStore.GetAll()

	return s.ExecuteProbes(ctx, probes...)
}

func (s service) ExecuteProbes(ctx context.Context, probes ...Probe) ([]ExecutionResult, error) {
	executionResults := s.executeProbes(ctx, probes)

	go s.metricsService.UpdateGauge(executionResults...)

	return executionResults, nil
}

func (s service) executeProbes(ctx context.Context, probes []Probe) []ExecutionResult {
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
				r.Err = err
				r.Probe.Health = UnhealthyStatus
			} else {
				r.Probe.Health = HealthyStatus
			}

			c <- r
		}(p)
	}

	executionResults := make([]ExecutionResult, 0, len(probes))
	for executionResult := range c {
		executionResults = append(executionResults, executionResult)
	}

	return executionResults
}

func (s service) ExecuteProbesByKind(ctx context.Context, kind ProbeKind) ([]ExecutionResult, error) {
	probes := make([]Probe, 0)

	if kind == CustomProbeKind {
		probes = s.probeStore.GetAll()
	} else {
		probes = s.probeStore.GetByKind(kind)
	}

	executionResults, err := s.ExecuteProbes(ctx, probes...)
	if err != nil {
		return nil, err
	}

	return executionResults, nil
}

func NewService(probeStore ProbeStore, metricsService MetricsService) Service {
	s := &service{
		metricsService: metricsService,
		probeStore:     probeStore,
	}

	return s
}
