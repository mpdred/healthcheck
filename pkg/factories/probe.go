package factories

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

	"github.com/mpdred/healthcheck/v2/pkg/healthcheck"
	"github.com/pkg/errors"
)

// ProbeBuilder creates a Probe.
// Has some predefined ProbeCheckFn(s),
// for which it will set a predefined name if not otherwise specified by the user.
//
// It has a default timeout for the predefined checks.
type ProbeBuilder interface {
	WithKind(k healthcheck.ProbeKind) ProbeBuilder

	// WithName sets a friendly name for the probe.
	WithName(n string) ProbeBuilder

	// WithCustomCheck allows you to define your own function that is to be executed.
	WithCustomCheck(fn healthcheck.ProbeCheckFn) ProbeBuilder

	WithDatabaseConnectionCheck(database *sql.DB) ProbeBuilder
	WithDNSResolveCheck(host string) ProbeBuilder
	WithHTTPGetCheck(url string) ProbeBuilder
	WithTCPDialWithTimeoutCheck(address string) ProbeBuilder

	// Build the probe as requested.
	//
	// The ProbeKind is set to CustomProbeKind by default.
	//
	// Note: No checks are performed, so it allows for objects with undefined fields.
	Build() healthcheck.Probe

	// MustBuild uses Build to build the probe as requested,
	// and panic if there are any fields with undefined fields.
	//
	// The ProbeKind is set to CustomProbeKind by default.
	MustBuild() healthcheck.Probe

	// BuildLivenessProbe creates a Probe that always executes without errors.
	//
	// This can be used for readiness checks in your APIs.
	BuildLivenessProbe() healthcheck.Probe

	// BuildDeadmansSnitch creates a Probe that always returns an error.
	//
	// Usually an alert is created for its absence.
	BuildDeadmansSnitch() healthcheck.Probe

	// BuildForComponents creates probes base on the map's boolean values.
	BuildForComponents(kind healthcheck.ProbeKind, componentsStatusMap map[string]bool) []healthcheck.Probe
}

type probeBuilder struct {
	defaultTimeout time.Duration
	probe          *healthcheck.Probe
}

func NewProbeBuilder() ProbeBuilder {
	const defaultTimeout = 5 * time.Second

	b := probeBuilder{
		defaultTimeout: defaultTimeout,
		probe:          &healthcheck.Probe{},
	}

	return &b
}

func (b *probeBuilder) WithKind(k healthcheck.ProbeKind) ProbeBuilder {
	b.probe.Kind = k

	return b
}

func (b *probeBuilder) WithName(n string) ProbeBuilder {
	b.probe.Name = n

	return b
}

func (b *probeBuilder) WithCustomCheck(fn healthcheck.ProbeCheckFn) ProbeBuilder {
	b.probe.CheckFn = fn

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

	b.probe.CheckFn = fn

	if strings.TrimSpace(b.probe.Name) == "" {
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

	b.probe.CheckFn = fn

	if strings.TrimSpace(b.probe.Name) == "" {
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
			return errors.Wrapf(healthcheck.ErrCheckFailed, "status code: %d", resp.StatusCode)
		}

		return nil
	}

	b.probe.CheckFn = fn

	if strings.TrimSpace(b.probe.Name) == "" {
		const defaultName = "http get"
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

	b.probe.CheckFn = fn

	if strings.TrimSpace(b.probe.Name) == "" {
		const defaultName = "tcp dial"
		b.WithName(defaultName)
	}

	return b
}

func (b *probeBuilder) Build() healthcheck.Probe {
	b.probe.Name = strings.TrimSpace(b.probe.Name)

	if strings.EqualFold(string(b.probe.Kind), "") {
		b.probe.Kind = healthcheck.CustomProbeKind
	}

	return *b.probe
}

func (b *probeBuilder) MustBuild() healthcheck.Probe {
	p := b.Build()

	if strings.TrimSpace(string(p.Kind)) == "" {
		panic("no probe kind")
	}

	if strings.TrimSpace(p.Name) == "" {
		panic("no probe name")
	}

	if p.CheckFn == nil {
		panic("no probe check function")
	}

	return p
}

func (b *probeBuilder) BuildLivenessProbe() healthcheck.Probe {
	const defaultName = "liveness"
	fn := func(context.Context) error { return nil }

	probe := b.WithName(defaultName).
		WithCustomCheck(fn).
		WithKind(healthcheck.LivenessProbeKind).
		MustBuild()

	return probe
}

func (b *probeBuilder) BuildDeadmansSnitch() healthcheck.Probe {
	const defaultName = "dead man's snitch"
	fn := func(context.Context) error { return errors.New(defaultName) }

	probe := b.WithName(defaultName).
		WithCustomCheck(fn).
		WithKind(healthcheck.LivenessProbeKind).
		MustBuild()

	return probe
}

func (b *probeBuilder) BuildForComponents(kind healthcheck.ProbeKind, componentsStatusMap map[string]bool) []healthcheck.Probe {
	probes := make([]healthcheck.Probe, 0)

	for component := range componentsStatusMap {
		c := component

		fn := func(ctx context.Context) error {
			if !componentsStatusMap[c] {
				return errors.New("readiness for component set to 'false'")
			}

			return nil
		}

		p := NewProbeBuilder().
			WithName(fmt.Sprintf("component %s", component)).
			WithKind(kind).
			WithCustomCheck(fn).
			Build()

		probes = append(probes, p)
	}

	return probes
}
