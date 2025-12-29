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

// HandleDelete is a generic helper for DELETE endpoints that:
// 1. Calls the controller to delete the resource
// 2. Handles not found errors with 404
// 3. Handles unauthorized (401), forbidden (403), and conflict (409) errors
// 4. Handles other errors with 500
// 5. Returns 202 Accepted on success
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
		logger.ErrorContext(r.Context(), "failed to delete", slog.Any("error", err))
		status, message := apierr.ModelToHTTPError(err)
		http.Error(w, message, status)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
