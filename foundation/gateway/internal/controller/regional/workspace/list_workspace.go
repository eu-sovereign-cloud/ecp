package workspace

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes/labels"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type ListWorkspace struct {
	Logger *slog.Logger
	Repo   port.ReaderRepo[*regional.WorkspaceDomain]
}

func (c *ListWorkspace) Do(ctx context.Context, params model.ListParams) ([]*regional.WorkspaceDomain, *string, error) {
	// Work on a local copy so we don't mutate the caller's params.
	lp := params

	// If tenant is provided and workspace is empty, ensure the selector includes the tenant label as workspaces are to be listed from the internal tenant label.
	if lp.Scope.Tenant != "" {
		tenantSel := fmt.Sprintf("%s=%s", labels.InternalTenantLabel, lp.Scope.Tenant)
		lp.Scope.Tenant = ""
		if lp.Selector != "" {
			lp.Selector = tenantSel + "," + lp.Selector
		} else {
			lp.Selector = tenantSel
		}
	}

	var domains []*regional.WorkspaceDomain

	skipToken, err := c.Repo.List(ctx, lp, &domains)
	if err != nil {
		return nil, nil, err
	}
	return domains, skipToken, nil
}
