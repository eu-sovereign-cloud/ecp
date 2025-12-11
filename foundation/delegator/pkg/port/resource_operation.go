package port

import (
	"context"

	gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

// ConditionFunc should detect if a given resource give condition for some
// operation by evaluating and comparing its desired state (spec) to its
// current state (status).
//
// Scope: Common Delegator Domain
type ConditionFunc[T gateway_port.NamespacedResource] func(resource T) bool

// PluginDelegateFunc should delegate to the CSP adapter the responsibility to
// perform an operation which the condition was previously detected by the
// proper ConditionFunc.
//
// Scope: CSP Specific Plugin
type PluginDelegateFunc[T gateway_port.NamespacedResource] func(ctx context.Context, resource T) error

// PropagateSucessFunc should propagate the results of a succeed operation
// (which was previously detected by a ConditionFunc and delegated to a
// PluginDelegateFunc) from its desired state (spec) to its current state
// (status).
//
// Scope: Common Delegator Domain
type PropagateSucessFunc[T gateway_port.NamespacedResource] func(resource T)

// PropagateFailureFunc should propagate the results of a failed operation
// (which was previously detected by the ConditionFunc mathod and delegated
// to the PluginDelegateFunc method) from its desired state (spec) to its
// current state (status).
//
// Scope: Common Delegator Domain
type PropagateFailureFunc[T gateway_port.NamespacedResource] func(resource T, err error)

// SetStateFunc should be used to set the final state of the resource in the
// repository.
//
// In most cases it will point to one of the gateway_port.Writer
// implementation methods: Create(...), Update(...) or Delete(...).
//
// Scope: gateway_port.Writer
type SetStateFunc[T gateway_port.NamespacedResource] func(ctx context.Context, resource T) error

// ResourceOperation encapsulate the elements which allows 1) to detect a
// condition, 2) delegate the operationto the CSP adapter, 3) propagate the
// success results, and 4) propagate failure results.
//
// It is intended to be used in the context of the following logic:
//
//	...
//	if operation.GiveCondition(resource) {
//		if err := operation.Delegate(ctx, resource); err != nil {
//			operation.PropagateFailure(resource, err)
//			return operation.SetStateFailure(ctx, resource)
//		}
//
//		operation.PropagateSucess(resource)
//		return operation.SetStateSucess(ctx, resource)
//	}
//	...
type ResourceOperation[T gateway_port.NamespacedResource] interface {
	// GiveCondition should detect if a given resource give condition for some
	// operation by evaluating and comparing its desired state (spec) to its
	// current state (status).
	//
	// Scope: Common Delegator Domain
	GiveCondition(resource T) bool

	// Delegate should delegate to the CSP adapter the responsibility to
	// perform an operation which the condition was previously detected by the
	// GiveCondition method.
	//
	// It is equivalent to the "Plugin.Do(...) error" method in the proposal.
	//
	// Scope: CSP Specific Plugin
	Delegate(ctx context.Context, resource T) error

	// PropagateSuccess should propagate the results of a succeed operation
	// (which was previously detected by the GiveCondition mathod and delegated
	// to the Delegate method) from its desired state (spec) to its current
	// state (status).
	//
	// Scope: Common Delegator Domain
	PropagateSuccess(resource T)

	// SetStateSucess set the final state of the resource in the repository for
	// successful operations.
	//
	// In most cases it will point to one of the gateway_port.Writer
	// implementation methods: Create(...), Update(...) or Delete(...).
	//
	// Scope: gateway_port.Writer
	SetStateSucess(ctx context.Context, resource T) error

	// PropagateFailure should propagate the results of a failed operation
	// (which was previously detected by the GiveCondition mathod and delegated
	// to the Delegate method) from its desired state (spec) to its current
	// state (status).
	//
	// Scope: Common Delegator Domain
	PropagateFailure(resource T, err error)

	// SetStateFailure set the final state of the resource in the repository
	// for failed operations.
	//
	// In most cases it will point to one of the gateway_port.Writer
	// implementation methods: Create(...), Update(...) or Delete(...).
	//
	// Scope: gateway_port.Writer
	SetStateFailure(ctx context.Context, resource T) error
}

// GenericResourceOperation can be used to compose specific operation plugins
// by combining existing functions.
type GenericResourceOperation[T gateway_port.NamespacedResource] struct {
	condition        ConditionFunc[T]
	pluginDelegate   PluginDelegateFunc[T]
	propagateSucess  PropagateSucessFunc[T]
	setStateSucess   SetStateFunc[T]
	propagateFailure PropagateFailureFunc[T]
	setStateFailure  SetStateFunc[T]
}

var _ ResourceOperation[gateway_port.NamespacedResource] = &GenericResourceOperation[gateway_port.NamespacedResource]{}

func NewResourceOperation[T gateway_port.NamespacedResource](
	condition ConditionFunc[T],
	propagateSucess PropagateSucessFunc[T],
	setStateSucess SetStateFunc[T],
	propagateFailure PropagateFailureFunc[T],
	setStateFailure SetStateFunc[T],
) *GenericResourceOperation[T] {
	return &GenericResourceOperation[T]{
		condition:        condition,
		propagateSucess:  propagateSucess,
		setStateSucess:   setStateSucess,
		propagateFailure: propagateFailure,
		setStateFailure:  setStateFailure,
	}
}

func NewResourceOperationWithPluginDelegate[T gateway_port.NamespacedResource](
	condition ConditionFunc[T],
	pluginDelegate PluginDelegateFunc[T],
	propagateSucess PropagateSucessFunc[T],
	setStateSucess SetStateFunc[T],
	propagateFailure PropagateFailureFunc[T],
	setStateFailure SetStateFunc[T],
) *GenericResourceOperation[T] {
	return &GenericResourceOperation[T]{
		condition:        condition,
		pluginDelegate:   pluginDelegate,
		propagateSucess:  propagateSucess,
		setStateSucess:   setStateSucess,
		propagateFailure: propagateFailure,
		setStateFailure:  setStateFailure,
	}
}

func (o *GenericResourceOperation[T]) SetPluginDelegate(pluginDelegate PluginDelegateFunc[T]) {
	o.pluginDelegate = pluginDelegate
}

func (o *GenericResourceOperation[T]) GiveCondition(resource T) bool {
	return o.condition(resource)
}

func (o *GenericResourceOperation[T]) Delegate(ctx context.Context, resource T) error {
	return o.pluginDelegate(ctx, resource)
}

func (o *GenericResourceOperation[T]) PropagateSuccess(resource T) {
	o.propagateSucess(resource)
}

func (o *GenericResourceOperation[T]) SetStateSucess(ctx context.Context, resource T) error {
	return o.setStateSucess(ctx, resource)
}

func (o *GenericResourceOperation[T]) PropagateFailure(resource T, err error) {
	o.propagateFailure(resource, err)
}

func (o *GenericResourceOperation[T]) SetStateFailure(ctx context.Context, resource T) error {
	return o.setStateFailure(ctx, resource)
}

// NoopSetStateFunc creates a SetStateFunc for the given type which does
// nothing.
func NoopSetStateFunc[T gateway_port.NamespacedResource]() SetStateFunc[T] {
	return func(_ context.Context, _ T) error {
		return nil
	}
}
