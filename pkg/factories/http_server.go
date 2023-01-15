package factories

import (
	"context"
	"fmt"
	"net"
	"net/http"
)

type HTTPServerBuilder interface {
	WithPort(port int) HTTPServerBuilder
	WithHandler(handler http.Handler) HTTPServerBuilder
	WithBaseContext(baseContextFn func(net.Listener) context.Context) HTTPServerBuilder
	Build(ctx context.Context) *http.Server
}

type serverBuilder struct {
	httpHandler http.Handler
	port        int
	baseContext func(l net.Listener) context.Context
}

func (s *serverBuilder) WithPort(port int) HTTPServerBuilder {
	s.port = port

	return s
}

func (s *serverBuilder) WithHandler(handler http.Handler) HTTPServerBuilder {
	s.httpHandler = handler

	return s
}

func (s *serverBuilder) WithBaseContext(baseContextFn func(net.Listener) context.Context) HTTPServerBuilder {
	s.baseContext = baseContextFn

	return s
}

func (s *serverBuilder) Build(ctx context.Context) *http.Server {
	if s.port == 0 {
		const defaultPort = 5090
		s.WithPort(defaultPort)
	}

	if s.httpHandler == nil {
		mux := http.NewServeMux()
		s.WithHandler(mux)
	}

	if s.baseContext == nil {
		type serverAddrType string
		serverAddr := serverAddrType("serverAddr")

		fn := func(l net.Listener) context.Context {
			ctx = context.WithValue(ctx, serverAddr, l.Addr().String())
			return ctx
		}

		s.WithBaseContext(fn)
	}

	httpServer := &http.Server{
		Addr:        fmt.Sprintf(":%d", s.port),
		Handler:     s.httpHandler,
		BaseContext: s.baseContext,
	}

	return httpServer
}

func NewServerBuilder() HTTPServerBuilder {
	b := &serverBuilder{}

	return b
}
