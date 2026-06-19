package delegated

import (
	"context"

	delegator "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"

	resolver_bypass "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/resolver"
	converter_port "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/converter"
	delegated_port "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/delegated"
	mutator_port "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/mutator"
	repository_port "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/repository"
	resolver_port "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/resolver"
)

// GenericDelegated acts as a standard framework for all delegated resource
// operations for the Aruba Plugin.
//
// It defines the data structures used throughout the process using the
// following type parameters:
//
//   - S: The original SECA resource type.
//   - SB: The structure which should hold the dependencies for the SECA resource.
//   - AB: The structure which should hold all the Aruba resource we need to the operation.
//
// The elements intended to handle the SB and AB types can be composed by
// arrangements of elements capable to handle its underlying types.
//
// Usage Example:
//
//	type MyResourceOperation struct {
//		*delegated.GenericDelegated[*v1.MySECAResource, *MySECABundle, *MyArubaBundle]
//		repository *MyArubaRepository
//		// other pertinent components
//	}
//
//	func NewMyResourceOperation(repo *MyArubaRepository) *MyResourceOperation {
//		return &MyResourceOperation{
//			GenericDelegated: delegated.NewDelegated(
//				resolveSECAFunc,
//				convertFunc,
//				resolveArubaFunc,
//				mutateFunc,
//				repo.Propagate,
//				conditionFunc,
//				repo.WaitUntil,
//			),
//			repository: repo,
//		}
//	}
type GenericDelegated[
	S persistence.IdentifiableResource, // SECA resource type
	SB any, // SECA bundle type
	AB any, // Aruba bundle type
] struct {
	resolveSECA  resolver_port.ResolveDependenciesFunc[S, SB]
	convert      converter_port.ConvertFunc[SB, AB]
	resolveAruba resolver_port.ResolveDependenciesFunc[AB, AB]
	mutate       mutator_port.MutateFunc[AB, SB]
	propagate    repository_port.CLUDFunc[AB]
	check        delegated_port.CheckFunc[SB, AB]
}

var _ delegated_port.Delegated[persistence.IdentifiableResource] = (*GenericDelegated[persistence.IdentifiableResource, any, any])(nil)

// NewDelegated creates a new instance of GenericDelegated with the provided
// handler functions.
func NewDelegated[S persistence.IdentifiableResource, SB any, AB any](
	resolveSECAFunc resolver_port.ResolveDependenciesFunc[S, SB],
	convertFunc converter_port.ConvertFunc[SB, AB],
	resolveArubaFunc resolver_port.ResolveDependenciesFunc[AB, AB],
	mutateFunc mutator_port.MutateFunc[AB, SB],
	propagateFunc repository_port.CLUDFunc[AB],
	checkFunc delegated_port.CheckFunc[SB, AB],
) *GenericDelegated[S, SB, AB] {
	return &GenericDelegated[S, SB, AB]{
		resolveSECA:  resolveSECAFunc,
		convert:      convertFunc,
		resolveAruba: resolveArubaFunc,
		mutate:       mutateFunc,
		propagate:    propagateFunc,
		check:        checkFunc,
	}
}

// NewStraightDelegated creates a new instance of GenericDelegated, for
// resources which do not need bundle, with the provided handler functions.
func NewStraightDelegated[S persistence.IdentifiableResource, A any](
	convertFunc converter_port.ConvertFunc[S, A],
	mutateFunc mutator_port.MutateFunc[A, S],
	propagateFunc repository_port.CLUDFunc[A],
	checkFunc delegated_port.CheckFunc[S, A],
) *GenericDelegated[S, S, A] {
	return &GenericDelegated[S, S, A]{
		resolveSECA:  resolver_bypass.BypassResolveDependenciesFunc[S],
		convert:      convertFunc,
		resolveAruba: resolver_bypass.BypassResolveDependenciesFunc[A],
		mutate:       mutateFunc,
		propagate:    propagateFunc,
		check:        checkFunc,
	}
}

// Do executes the delegated resource operation following the standard steps:
// 1. Resolve SECA dependencies.
// 2. Convert SECA bundle to Aruba bundle.
// 3. Resolve Aruba dependencies.
// 4. Check whether the desired state is already reached; if so, return nil.
// 5. Otherwise mutate the Aruba resources.
// 6. Propagate changes to the Aruba Cloud.
// 7. Report that the operation is still in progress (ErrStillProcessing) if the check does not pass yet, so the reconciler can requeue and check again later without blocking.
func (d *GenericDelegated[S, SB, AB]) Do(ctx context.Context, resource S) error {
	// 1. Resolve SECA-level dependencies for referenced objects in the Aruba
	// domain.
	//
	// In that step, the plugin handler should be able to check if the SECA
	// resource passed as a parameter is linked to all the mandatory
	// dependencies in the SECA context, but not only for the dependencies
	// defined by the SECA specs, but also for dependencies defined by Aruba
	// internal requirements.
	secaBundle, err := d.resolveSECA(ctx, resource)
	if err != nil {
		return err
	}

	// 2. Convert the SECA Business Models to the equivalent Aruba resources
	// representation.
	//
	// In that step, all the SECA resources, including the one passed as
	// parameter and also the other fetched in the previous step, should be
	// converted to equivalent Aruba resource representations.
	arubaBundle, err := d.convert(secaBundle)
	if err != nil {
		return err
	}

	// 3. Resolve Aruba dependencies for referenced objects in the Aruba domain.
	//
	// In that step, the plugin should be able to locate and retrieve all the
	// already existing Aruba resources necessary to perform the intended
	// operation, including those that should be mutated, and those that should
	// only be read to retrieve some information.
	arubaBundle, err = d.resolveAruba(ctx, arubaBundle)
	if err != nil {
		return err
	}

	// 4. Check whether the desired state has already been reached.
	//
	// Unlike a blocking wait, this step inspects the affected Aruba resources
	// once. When every required resource is already present in its target
	// state, there is nothing left to do and the operation is complete.
	done, err := d.check(ctx, secaBundle, arubaBundle)
	if err != nil {
		return err
	}
	if done {
		return nil
	}

	// 5. Mutate the Aruba resources according the received specs.
	//
	// In that step, the plugin handler should be able to perform all mutations
	// on the Aruba resources that are necessary to achieve the desired state
	// according to the specs of the SECA resource passed as a parameter.
	if err := d.mutate(arubaBundle, secaBundle); err != nil {
		return err
	}

	// 6. Trigger the required action to the Aruba Provisioner.
	//
	// In that step, the plugin handler should be able to trigger all actions
	// necessary to achieve the wanted state. In practice, it means to handle
	// the Aruba resources in the Kubernetes cluster.
	//
	// As the action may be (re)issued on every pass until the check above
	// passes, propagate must be idempotent.
	if err := d.propagate(ctx, arubaBundle); err != nil {
		return err
	}

	// 7. Report that the operation is still in progress.
	//
	// The plugin calls are non-blocking: rather than waiting in-process, we
	// return ErrStillProcessing so the reconciler requeues and checks again on
	// a later pass without holding the worker.
	return delegator.ErrStillProcessing
}
