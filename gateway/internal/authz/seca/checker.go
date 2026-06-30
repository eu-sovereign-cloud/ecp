package seca

import (
	"context"
	"fmt"
	"log/slog"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
)

// Checker is the reader-backed SECA RBAC implementation of authzport.Checker.
//
// On each Authorize call it:
//  1. Lists all RoleAssignments in the claim's tenant namespace.
//  2. Lists all Roles referenced by those assignments.
//  3. Calls Evaluate (pure, no I/O) to determine if the claim is permitted.
//
// This is requirement 2.2 from the design. For the cached variant (2.3) see CachedChecker.
type Checker struct {
	roleReader       persistence.ReaderRepo[*roledom.Role]
	assignmentReader persistence.ReaderRepo[*radom.RoleAssignment]
	log              *slog.Logger
}

// NewChecker creates a Checker backed by Kubernetes reader adapters.
// The roleReader and assignmentReader are typically constructed with
// k8sadapter.NewReaderAdapter in the gateway server command.
func NewChecker(
	roleReader persistence.ReaderRepo[*roledom.Role],
	assignmentReader persistence.ReaderRepo[*radom.RoleAssignment],
	log *slog.Logger,
) *Checker {
	return &Checker{
		roleReader:       roleReader,
		assignmentReader: assignmentReader,
		log:              log,
	}
}

// Authorize implements authzport.Checker.
// Returns nil when the claim is permitted, kernel.ErrForbidden when denied.
func (c *Checker) Authorize(ctx context.Context, claim authzport.AuthorizationClaim) error {
	rolesByName, assignments, err := c.load(ctx, claim.Tenant)
	if err != nil {
		c.log.ErrorContext(ctx, "seca rbac: failed to load policy data", slog.Any("error", err))
		return kernel.ErrForbidden
	}

	if Evaluate(claim, rolesByName, assignments) {
		return nil
	}
	return kernel.ErrForbidden
}

// load fetches roles and assignments for the given tenant namespace.
func (c *Checker) load(ctx context.Context, tenant string) (map[string]*roledom.Role, []*radom.RoleAssignment, error) {
	tenantScope := resource.ListParams{Scope: resource.Scope{Tenant: tenant}}

	var roleList []*roledom.Role
	if _, err := c.roleReader.List(ctx, tenantScope, &roleList); err != nil {
		return nil, nil, fmt.Errorf("list roles: %w", err)
	}

	var assignmentList []*radom.RoleAssignment
	if _, err := c.assignmentReader.List(ctx, tenantScope, &assignmentList); err != nil {
		return nil, nil, fmt.Errorf("list role assignments: %w", err)
	}

	rolesByName := make(map[string]*roledom.Role, len(roleList))
	for _, r := range roleList {
		rolesByName[r.GetName()] = r
	}

	return rolesByName, assignmentList, nil
}
