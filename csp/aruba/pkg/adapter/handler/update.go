package handler

import (
	"context"
	"slices"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/repository"
)

// Updating an Aruba resource means updating its tags, and nothing else.
//
// Tags are the only field the Aruba API lets an update change on every resource type, and on a VPC
// or a SecurityGroup they are the only mutable field at all. Region, zone, flavor, CIDR and every
// reference are fixed at creation; the operator screens a change to any of them through its
// HasDeniedChanges check and fails the resource rather than calling the CMP. So the handlers below
// re-apply tags and leave the rest of the spec alone - re-applying a converted spec wholesale
// would hand the operator exactly the denied changes it is looking for.
//
// A SECA resource's tags come from its labels (see converter.ArubaTags), so this is what carries a
// label edit through to the provider.

// syncSpec reads the live Aruba object and writes it back only when apply reports a real
// difference. Update is called on every reconcile of an active resource, and an unconditional
// write would churn the Aruba CR - which the operator watches - on every pass.
//
// apply is handed the live object and returns whether it changed anything.
// A missing Aruba counterpart is reported, not swallowed. The SECA resource is active, so its
// counterpart existed at some point and a brief gap really can be timing - but ErrStillProcessing
// tells the reconciler to requeue and leave status untouched, and nothing bounds how long that can
// go on. An object deleted out of band never comes back, so the resource would requeue every five
// minutes forever while reporting itself active, with nothing in the API, its conditions, or the
// logs saying otherwise. Reported as a plain error it is still retried - so genuine timing still
// resolves itself, and the condition clears when it does - but a permanent gap is visible.
func syncSpec[T, L any](ctx context.Context, repo repository.Repository[T, L], obj T, apply func(T) bool) error {
	if err := repo.Load(ctx, obj); err != nil {
		return err
	}

	if !apply(obj) {
		return nil
	}

	return repo.Update(ctx, obj)
}

// syncTags is syncSpec for tags, which is every handler here bar the workspace's - see
// workspace.go, where a Project's description travels alongside them.
func syncTags[T, L any](
	ctx context.Context,
	repo repository.Repository[T, L],
	obj T,
	desired []string,
	tagsOf func(T) *[]string,
) error {
	// Load overwrites obj with the live object, taking the desired tags with it.
	desired = slices.Clone(desired)

	return syncSpec(ctx, repo, obj, func(obj T) bool {
		current := tagsOf(obj)
		if slices.Equal(*current, desired) {
			return false
		}

		*current = desired

		return true
	})
}
