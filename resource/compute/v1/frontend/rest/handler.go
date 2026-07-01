// Package rest provides the HTTP handlers for the seca.compute API group.
// The compute group is not yet implemented: every method returns HTTP 501.
// The package exists so the gateway can register a complete seca.compute
// provider (with the auth middleware chain), mirroring the other groups.
package rest

import (
	"log/slog"

	sdkcompute "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.compute.v1"
)

// Handler implements sdkcompute.ServerInterface for the seca.compute provider.
// It is currently a stub — all methods return HTTP 501 Not Implemented.
type Handler struct {
	Logger *slog.Logger
}

var _ sdkcompute.ServerInterface = (*Handler)(nil)
