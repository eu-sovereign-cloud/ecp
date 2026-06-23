package rest

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
)

// Deleter defines the interface for controller Delete operations.
type Deleter interface {
	Do(ctx context.Context, resource persistence.IdentifiableResource) error
}

// HandleDelete is a generic helper for DELETE endpoints that writes 202 Accepted on success.
func HandleDelete(
	w http.ResponseWriter,
	r *http.Request,
	logger *slog.Logger,
	ir persistence.IdentifiableResource,
	deleter Deleter,
) {
	logger = logger.With("name", ir.GetName(), "tenant", ir.GetTenant(), "workspace", ir.GetWorkspace())

	if err := deleter.Do(r.Context(), ir); err != nil {
		WriteErrorResponse(w, r, logger, err)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
