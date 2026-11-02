// Package observer sets up logs, traces and metrics for a service with one call.
package observer

import (
	"context"
	"errors"

	"github.com/goleggo/observer/config"
	"github.com/goleggo/observer/log"
	"github.com/goleggo/observer/metrics"
	"github.com/goleggo/observer/trace"
)

// Setup installs the slog default, the propagator, and the tracer and meter providers. The returned shutdown flushes
// and stops all of them; call it once on exit. If Setup fails, it has already stopped whatever it started.
func Setup(ctx context.Context, cfg config.Config) (shutdown func(context.Context) error, err error) {
	var stops []func(context.Context) error
	shutdown = func(ctx context.Context) error {
		var errs []error
		for i := len(stops) - 1; i >= 0; i-- {
			errs = append(errs, stops[i](ctx))
		}
		stops = nil
		return errors.Join(errs...)
	}
	fail := func(err error) (func(context.Context) error, error) {
		return nil, errors.Join(err, shutdown(ctx))
	}

	lp, err := log.SetupOTELLogger(ctx, cfg.Log, cfg.OTEL)
	if err != nil {
		return fail(err)
	}
	if lp == nil {
		log.SetupLogger(cfg.Log)
	} else {
		stops = append(stops, lp.Shutdown)
	}

	tp, err := trace.SetupTrace(ctx, cfg.OTEL)
	if err != nil {
		return fail(err)
	}
	if tp != nil {
		stops = append(stops, tp.Shutdown)
	}

	mp, err := metrics.SetupMetrics(ctx, cfg.OTEL)
	if err != nil {
		return fail(err)
	}
	if mp != nil {
		stops = append(stops, mp.Shutdown)
	}
	return shutdown, nil
}
