package handler

import (
	"context"
	"slices"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	backend "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"

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

// syncTags brings one Aruba resource's tags in line with the SECA labels it was converted from.
//
// The live object is read first and written only on a real difference: Update is called on every
// reconcile of an active resource, and an unconditional write would churn the Aruba CR - which the
// operator watches - on every pass.
func syncTags[T, L any](
	ctx context.Context,
	repo repository.Repository[T, L],
	obj T,
	desired []string,
	tagsOf func(T) *[]string,
) error {
	// Load overwrites obj with the live object, taking the desired tags with it.
	desired = slices.Clone(desired)

	if err := repo.Load(ctx, obj); err != nil {
		if apierrors.IsNotFound(err) {
			// The SECA resource is active, so its Aruba counterpart existed at some point. Treat a
			// gap as timing rather than as a failure and look again on the next pass.
			return backend.ErrStillProcessing
		}

		return err
	}

	current := tagsOf(obj)
	if slices.Equal(*current, desired) {
		return nil
	}

	*current = desired

	return repo.Update(ctx, obj)
}

// syncProject is syncTags for the Aruba Project, which carries a second mutable field: an Aruba
// project has a description, and the SECA workspace spec supplies one, so the two are reconciled
// together rather than leaving the description frozen at whatever creation set.
func syncProject(
	ctx context.Context,
	repo repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList],
	prj *v1alpha1.Project,
	desiredTags []string,
	desiredDescription string,
) error {
	desiredTags = slices.Clone(desiredTags)

	if err := repo.Load(ctx, prj); err != nil {
		if apierrors.IsNotFound(err) {
			return backend.ErrStillProcessing
		}

		return err
	}

	if slices.Equal(prj.Spec.Tags, desiredTags) && prj.Spec.Description == desiredDescription {
		return nil
	}

	prj.Spec.Tags = desiredTags
	prj.Spec.Description = desiredDescription

	return repo.Update(ctx, prj)
}
