package rest

import (
	"context"

	frest "github.com/eu-sovereign-cloud/ecp/framework/frontend/rest"
	persistencepkg "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// activeOnWrite wraps a spec write (Create or Update) so it also stamps an active
// status subresource. Role and RoleAssignment are managed entirely by the ECP
// control plane — there is no plugin to reconcile them — so they are active the
// moment they are written. Status is a subresource, so the API server drops it on
// the spec write; markActive + UpdateStatus persist it in a second call.
type activeOnWrite[D persistencepkg.IdentifiableResource] struct {
	persist      func(context.Context, D) (*D, error)
	updateStatus func(context.Context, D) (*D, error)
	markActive   func(D)
}

func (a activeOnWrite[D]) Do(ctx context.Context, m D) (D, error) {
	var zero D
	saved, err := a.persist(ctx, m)
	if err != nil {
		return zero, err
	}
	a.markActive(*saved)
	withStatus, err := a.updateStatus(ctx, *saved)
	if err != nil {
		return zero, err
	}
	return *withStatus, nil
}

// activeCreator returns a Creator that persists the resource and marks it active.
func activeCreator[D persistencepkg.IdentifiableResource](repo persistencepkg.WriterRepo[D], markActive func(D)) frest.Creator[D] {
	return activeOnWrite[D]{persist: repo.Create, updateStatus: repo.UpdateStatus, markActive: markActive}
}

// activeUpdater returns an Updater that persists the resource and marks it active.
func activeUpdater[D persistencepkg.IdentifiableResource](repo persistencepkg.WriterRepo[D], markActive func(D)) frest.Updater[D] {
	return activeOnWrite[D]{persist: repo.Update, updateStatus: repo.UpdateStatus, markActive: markActive}
}

func markRoleActive(r *roledom.Role) {
	r.Status = &roledom.RoleStatus{}
	r.Status.PushCondition(commonbackend.ConditionFromState(commondomain.ResourceStateActive))
}

func markRoleAssignmentActive(ra *radom.RoleAssignment) {
	ra.Status = &radom.RoleAssignmentStatus{}
	ra.Status.PushCondition(commonbackend.ConditionFromState(commondomain.ResourceStateActive))
}
