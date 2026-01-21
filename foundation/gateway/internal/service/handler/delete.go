package handler

import (
	"context"
	"log/slog"
	"net/http"

	apierr "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/api/errors"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

// Deleter defines the interface for controller Delete operations
type Deleter interface {
	Do(ctx context.Context, resource port.IdentifiableResource) error
}

// HandleDelete is a generic helper for DELETE endpoints
func HandleDelete(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	ir port.IdentifiableResource,
	deleter Deleter,
) {
	logger = logger.With("name", ir.GetName(), "tenant", ir.GetTenant(), "workspace", ir.GetWorkspace())

	err := deleter.Do(r.Context(), ir)
	if err != nil {
		apierr.WriteErrorResponse(w, r, logger, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
