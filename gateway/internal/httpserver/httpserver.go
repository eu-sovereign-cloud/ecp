package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// DefaultShutdownTimeout is the drain budget for graceful Shutdown after the
// serve context is cancelled (SIGINT/SIGTERM). Keep below Kubernetes
// terminationGracePeriodSeconds (deployments use 30s).
const DefaultShutdownTimeout = 25 * time.Second

// Options defines the configuration for a new HTTP server.
type Options struct {
	Addr           string
	Handler        http.Handler
	Logger         *slog.Logger
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
	HeaderTimeout  time.Duration
	MaxHeaderBytes int
}

// New returns a configured *http.Server with sane defaults,
// overridden by any provided Options fields.
func New(opts Options) *http.Server {
	// provide defaults if not set
	if opts.ReadTimeout == 0 {
		opts.ReadTimeout = 30 * time.Second
	}
	if opts.WriteTimeout == 0 {
		opts.WriteTimeout = 60 * time.Second
	}
	if opts.IdleTimeout == 0 {
		opts.IdleTimeout = 120 * time.Second
	}
	if opts.HeaderTimeout == 0 {
		opts.HeaderTimeout = 10 * time.Second
	}
	if opts.MaxHeaderBytes == 0 {
		opts.MaxHeaderBytes = 1 << 20 // 1 MB
	}

	httpLogger := slog.NewLogLogger(opts.Logger.Handler(), slog.LevelInfo)

	return &http.Server{
		Addr:              opts.Addr,
		Handler:           opts.Handler,
		ReadTimeout:       opts.ReadTimeout,
		WriteTimeout:      opts.WriteTimeout,
		IdleTimeout:       opts.IdleTimeout,
		ReadHeaderTimeout: opts.HeaderTimeout,
		MaxHeaderBytes:    opts.MaxHeaderBytes,
		ErrorLog:          httpLogger,
	}
}

// Serve binds srv.Addr, serves until ctx is cancelled or Serve fails, then
// drains in-flight requests via Shutdown with DefaultShutdownTimeout.
//
// The listener is opened before the serve loop so Shutdown cannot race a still-
// unbound ListenAndServe (which would leave the process serving after "shutdown").
//
// When readiness is non-nil it is marked not-ready as soon as shutdown starts so
// kubelet readiness probes fail and the Service stops routing new traffic during
// the drain window.
func Serve(ctx context.Context, srv *http.Server, log *slog.Logger, readiness *Readiness) error {
	if srv == nil {
		return fmt.Errorf("http server is nil")
	}
	if log == nil {
		log = slog.Default()
	}

	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", srv.Addr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		// Serve always closes ln; Shutdown makes it return ErrServerClosed.
		errCh <- srv.Serve(ln)
	}()

	select {
	case err := <-errCh:
		if readiness != nil {
			readiness.Set(false)
		}
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		// Fail readiness before drain so load balancers stop sending work.
		if readiness != nil {
			readiness.Set(false)
		}
		log.Info("shutting down HTTP server", slog.Duration("timeout", DefaultShutdownTimeout))
		// WithoutCancel keeps ctx values but not cancellation — Shutdown needs its own budget
		// after the serve context is already done.
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), DefaultShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			// Force-close if drain budget is exhausted so Serve returns.
			_ = srv.Close()
			// Wait for the serve goroutine to end, so the listener is closed and
			// the port is free. The error is always ErrServerClosed here.
			<-errCh
			return fmt.Errorf("HTTP server shutdown: %w", err)
		}
		err := <-errCh
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
