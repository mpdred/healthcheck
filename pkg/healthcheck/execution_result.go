package healthcheck

type ExecutionResult struct {
	Probe Probe
	Err   error
}
