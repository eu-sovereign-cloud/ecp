package workspace

import (
	"context"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type CreateWorkspace struct {
	Logger *slog.Logger
	Repo   port.WriterRepo[*regional.WorkspaceDomain]
}

func (c *CreateWorkspace) Do(ctx context.Context, domain *regional.WorkspaceDomain) (*regional.WorkspaceDomain, error) {
	statusPending := regional.ResourceStateCreating // spec lifecycle diagram expects "Pending" but conformace tests expect "Creating"
	domain.Status = &regional.WorkspaceStatusDomain{StatusDomain: regional.StatusDomain{State: &statusPending}}

	result, err := c.Repo.Create(ctx, domain)
	if err != nil {
		return nil, err
	}
	return *result, nil
}
