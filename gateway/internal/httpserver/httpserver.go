package httpserver

import (
	"log/slog"
	"net/http"
	"time"
)

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
