package healthcheck

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/pkg/errors"
)

// ProbeBuilder creates a Probe.
// Has some built-in ProbeCheckFn,
// for which it will set a predefined name if not otherwise specified by the user.
//
// It has a default timeout for the predefined checks.
type ProbeBuilder interface {
	WithKind(k ProbeKind) ProbeBuilder
	WithName(n string) ProbeBuilder

	WithCustomCheck(fn ProbeCheckFn) ProbeBuilder

	WithDatabaseConnectionCheck(database *sql.DB) ProbeBuilder
	WithDNSResolveCheck(host string) ProbeBuilder
	WithHTTPGetCheck(url string) ProbeBuilder
	WithOpensearchConnectionCheck(client *opensearch.Client) ProbeBuilder
	WithRedisConnectionCheck(client *redis.Client) ProbeBuilder
	WithTCPDialWithTimeoutCheck(address string) ProbeBuilder

	// Build the probe as requested.
	//
	// The ProbeKind is set to Health by default.
	//
	// Note: No checks are performed, so it allows for objects with undefined fields.
	Build() *Probe

	// MustBuild uses Build to build the probe as requested,
	// and panic if there are any fields with undefined fields.
	//
	// The ProbeKind is set to Health by default.
	MustBuild() *Probe

	// BuildDeadmansSnitch creates a Probe that always executes without errors.
	//
	// Usually an alert is created for its absence.
	BuildDeadmansSnitch() *Probe
}

type probeBuilder struct {
	defaultTimeout time.Duration
	probe          *Probe
}

func NewProbeBuilder() ProbeBuilder {
	const defaultTimeout = 5 * time.Second

	b := probeBuilder{
		defaultTimeout: defaultTimeout,
		probe:          &Probe{},
	}

	return &b
}

func (b *probeBuilder) WithKind(k ProbeKind) ProbeBuilder {
	b.probe.kind = k

	return b
}

func (b *probeBuilder) WithName(n string) ProbeBuilder {
	b.probe.name = n

	return b
}

func (b *probeBuilder) WithCustomCheck(fn ProbeCheckFn) ProbeBuilder {
	b.probe.checkFn = fn

	return b
}

func (b *probeBuilder) WithDatabaseConnectionCheck(database *sql.DB) ProbeBuilder {
	fn := func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, b.defaultTimeout)
		defer cancel()

		if database == nil {
			return fmt.Errorf("database is nil")
		}

		return database.PingContext(ctx)
	}

	b.probe.checkFn = fn

	if strings.TrimSpace(b.probe.name) == "" {
		const defaultName = "sql database"
		b.WithName(defaultName)
	}

	return b
}

func (b *probeBuilder) WithDNSResolveCheck(host string) ProbeBuilder {
	resolver := net.Resolver{}

	fn := func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, b.defaultTimeout)
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

	b.probe.checkFn = fn

	if strings.TrimSpace(b.probe.name) == "" {
		const defaultName = "dns resolve"
		b.WithName(defaultName)
	}

	return b
}

func (b *probeBuilder) WithHTTPGetCheck(url string) ProbeBuilder {
	client := http.Client{
		Timeout: b.defaultTimeout,

		// don't follow redirects
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	fn := func(context.Context) error {
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

	b.probe.checkFn = fn

	if strings.TrimSpace(b.probe.name) == "" {
		const defaultName = "http get"
		b.WithName(defaultName)
	}

	return b
}

func (b *probeBuilder) WithOpensearchConnectionCheck(client *opensearch.Client) ProbeBuilder {
	fn := func(context.Context) error {
		_, err := client.Ping()

		return err
	}

	b.probe.checkFn = fn

	if strings.TrimSpace(b.probe.name) == "" {
		const defaultName = "opensearch"
		b.WithName(defaultName)
	}

	return b
}

func (b *probeBuilder) WithRedisConnectionCheck(client *redis.Client) ProbeBuilder {
	fn := func(ctx context.Context) error {
		out := client.Ping(ctx)

		return out.Err()
	}

	b.probe.checkFn = fn

	if strings.TrimSpace(b.probe.name) == "" {
		const defaultName = "redis"
		b.WithName(defaultName)
	}

	return b
}

func (b *probeBuilder) WithTCPDialWithTimeoutCheck(address string) ProbeBuilder {
	fn := func(context.Context) error {
		conn, err := net.DialTimeout("tcp", address, b.defaultTimeout)
		if err != nil {
			return err
		}

		return conn.Close()
	}

	b.probe.checkFn = fn

	if strings.TrimSpace(b.probe.name) == "" {
		const defaultName = "tcp dial"
		b.WithName(defaultName)
	}

	return b
}

func (b *probeBuilder) Build() *Probe {
	b.probe.name = strings.TrimSpace(b.probe.name)

	if strings.EqualFold(string(b.probe.kind), "") {
		b.probe.kind = Health
	}

	return b.probe
}

func (b *probeBuilder) MustBuild() *Probe {
	p := b.Build()

	if strings.TrimSpace(string(p.GetKind())) == "" {
		panic("no probe kind")
	}

	if strings.TrimSpace(p.GetName()) == "" {
		panic("no probe name")
	}

	if p.checkFn == nil {
		panic("no probe check function")
	}

	return p
}

func (b *probeBuilder) BuildDeadmansSnitch() *Probe {
	fn := func(context.Context) error { return nil }

	b.probe.checkFn = fn

	const defaultName = "dead man's snitch"
	b.WithName(defaultName)

	b.probe.kind = Liveness

	return b.probe
}
