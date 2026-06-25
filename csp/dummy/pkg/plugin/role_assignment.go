package plugin

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"time"

	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role-assignment/v1"
)

type RoleAssignment struct {
	logger *slog.Logger
}

func NewRoleAssignment(logger *slog.Logger) *RoleAssignment {
	return &RoleAssignment{logger: logger}
}

func (ra *RoleAssignment) Create(ctx context.Context, resource *radom.RoleAssignment) error {
	return simulateRA(ctx, "create", resource, roleAssignmentDelay(), ra.logger)
}

func (ra *RoleAssignment) Delete(ctx context.Context, resource *radom.RoleAssignment) error {
	return simulateRA(ctx, "delete", resource, roleAssignmentDelay(), ra.logger)
}

// roleAssignmentDelay returns the simulated latency of a role assignment operation.
func roleAssignmentDelay() time.Duration {
	const base int = 30

	variation := rand.IntN(60) //#nosec G404 -- math/rand/v2 is fine here: delay jitter is not security-sensitive

	return time.Duration(base+variation) * time.Second
}
