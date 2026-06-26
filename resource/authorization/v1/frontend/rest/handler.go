// Package rest provides REST↔domain conversion and HTTP handlers for the authorization API group.
package rest

import (
	"log/slog"

	sdkauth "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.authorization.v1"

	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
)

// Handler is the HTTP handler for the authorization API group.
// It owns the group's sdkauth.ServerInterface: role methods are implemented in
// role_handler.go, and role-assignment methods are stubbed in role_assignment_handler.go
// (pending full wiring in a future change).
type Handler struct {
	Reader persistencepkg.ReaderRepo[*roledom.Role]
	Writer persistencepkg.WriterRepo[*roledom.Role]
	Logger *slog.Logger
}

var _ sdkauth.ServerInterface = (*Handler)(nil)
