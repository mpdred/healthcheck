package healthcheck

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

type ProbeFnFactory interface {
	DatabasePingCheck(database *sql.DB, timeout time.Duration) ProbeCheckFn
	DNSResolveCheck(host string, timeout time.Duration) ProbeCheckFn
	HTTPGetCheck(url string, timeout time.Duration) ProbeCheckFn
	TCPDialWithTimeout(address string, timeout time.Duration) ProbeCheckFn
}

type defaultFactory struct{}

func (f defaultFactory) DatabasePingCheck(database *sql.DB, timeout time.Duration) ProbeCheckFn {
	fn := func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		if database == nil {
			return fmt.Errorf("database is nil")
		}

		return database.PingContext(ctx)
	}

	return fn
}

func (f defaultFactory) DNSResolveCheck(host string, timeout time.Duration) ProbeCheckFn {
	resolver := net.Resolver{}

	return func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		addrs, err := resolver.LookupHost(ctx, host)
		if err != nil {
			return err
		}

		if len(addrs) < 1 {
			return fmt.Errorf("could not resolve host")
		}

		return nil
	}
}

func (f defaultFactory) HTTPGetCheck(url string, timeout time.Duration) ProbeCheckFn {
	client := http.Client{
		Timeout: timeout,

		// don't follow redirects
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	fn := func(ctx context.Context) error {
		resp, err := client.Get(url)
		if err != nil {
			return err
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Println(err)
			}
		}(resp.Body)

		if resp.StatusCode >= http.StatusBadRequest {
			return errors.Wrapf(ErrCheckFailed, "status code: %d", resp.StatusCode)
		}

		return nil
	}

	return fn
}

func (f defaultFactory) TCPDialWithTimeout(address string, timeout time.Duration) ProbeCheckFn {
	fn := func(context.Context) error {
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			return err
		}

		return conn.Close()
	}

	return fn
}

func NewProbeFnFactory() ProbeFnFactory {
	var f ProbeFnFactory = &defaultFactory{}

	return f
}
