package workspace

import (
	"log/slog"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

type DeleteWorkspace struct {
	Logger *slog.Logger
	Repo   *port.WriterRepo[*regional.WorkspaceDomain]
}

func (c DeleteWorkspace) Do(domainWorkspace *regional.WorkspaceDomain) error {
	return nil
}
