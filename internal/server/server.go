package server

import (
	"log/slog"
	"net/http"
	"time"
)

// NewHTTPServer returns a configured *http.Server with defaults.
func NewHTTPServer(addr string, handler http.Handler, logger *slog.Logger) *http.Server {
	httpLogger := slog.NewLogLogger(logger.Handler(), slog.LevelInfo)

	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorLog:     httpLogger,
	}
}
