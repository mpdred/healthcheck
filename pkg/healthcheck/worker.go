package healthcheck

type Worker interface {
	Start()
	Stop()
}
