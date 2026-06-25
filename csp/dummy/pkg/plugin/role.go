package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role/v1"
)

// Role is the dummy CSP plugin for the role resource.
type Role struct {
	logger *slog.Logger
}

// NewRole creates a new dummy Role plugin.
func NewRole(logger *slog.Logger) *Role {
	return &Role{logger: logger}
}

// Create simulates the creation of a role.
func (ro *Role) Create(ctx context.Context, resource *roledom.Role) error {
	return simulateRole(ctx, "create", resource, roleDelay(), ro.logger)
}

// Delete simulates the deletion of a role.
func (ro *Role) Delete(ctx context.Context, resource *roledom.Role) error {
	return simulateRole(ctx, "delete", resource, roleDelay(), ro.logger)
}

// roleDelay returns the simulated latency of a role operation.
func roleDelay() time.Duration {
	const base int = 5

	variation := rand.IntN(10) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	return time.Duration(base+variation) * time.Second
}
