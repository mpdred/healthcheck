package healthcheck

import (
	"log"
	"net/http"

	"github.com/pkg/errors"
)

func StartHTTPServer(s *http.Server) {
	err := s.ListenAndServe()
	if err != nil {
		log.Printf("error listening for health check http server: %s\n", err)
	}
}

func StopHTTPServer(s *http.Server) {
	err := s.Close()
	if err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("error closing health check http server: %s\n", err)
		}
	}
}
