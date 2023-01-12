package healthcheck

type Worker interface {
	// Start the worker.
	Start()
	Stop()
}
