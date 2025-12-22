// Package delegated intends to be a standard framework for all delegated
// resource operations for the Aruba Plugin.
//
// SECA delegated resource operations (represented by the DelegatedFunc type)
// are intended to dispatch actions to be performed in the Aruba Cloud in order
// to achieve the state desired by the user, wait until the desired state is
// confirmed and then return.
//
// The SECA Delegator already contains the logic to know exactly which
// operation must be triggered. So the plugin handlers do not need to
// (re)implement such logic.
//
// The SECA Delegator is also intended to be able to check all dependencies the
// resource and/or its operations need.
//
// The input object is not intended to be mutated. The SECA Delegator also
// contains the logic to propagate both successful and failed operations.
//
// However, from the Aruba perspective, some other things should be important.
//
// For some resources there is no 1:1 matching between SECA and Aruba. The
// simplest case is when a single SECA resource should be mapped to multiple
// Aruba resources.
//
// However, for some cases we have the inverse situation: a single Aruba
// resource should be mapped to multiple SECA resources properly referenced
// between them.
//
// This situation requires special attention regarding dependency resolution:
//
// *Given* a user wants to create a SECA resource "A" which is represented by
// an Aruba resource "X" which should be mapped simultaneously to SECA
// resources "A", "B" and "C", *when* the user tries to create the resource
// "A", *then* the resource "A" will stay in a "pending" state while it waits
// the user to create also the resources "B" and "C" associated with each other
// and to "A".
//
// Then, in sequence:
//
// *Given* a user has created a SECA resource "A" which is represented by
// an Aruba resource "X" which should be mapped simultaneously to SECA resources
// "A", "B" and "C", *and* this resource "A" is in "pending" state while it is
// waiting for the companion resources "B" and "C", *when* the user
// creates the SECA resource "B" and "C" associated to the existing "A",
// *then* the plugin should properly map the existing Aruba resource "X" also
// to SECA resources "B" and "C", *and* all three resources can achieve the
// "active" state.
//
// In consequence, it's not possible for an already active SECA resource "A"
// associated to resource "B" and "C" to be unbound and re-associated to other
// instances of resources "B" and "C" linked to another instance of Aruba
// resource "X".
//
// That being said, the template method structure for a generic delegated
// resource operation handler consists of:
//
//  1. Resolve SECA-level dependencies for referenced objects in the Aruba
//     domain:
//
//     In that step, the plugin handler should be able to check if the SECA
//     resource passed as a parameter is linked to all the mandatory
//     dependencies in the SECA context, but not only for the dependencies
//     defined by the SECA specs, but also for dependencies defined by Aruba
//     internal requirements.
//
//     Based on the cases described above, in that step, when a plugin handler
//     is called to perform an operation on the SECA resource "A", it needs to
//     also locate and retrieve the SECA resources "B" and "C" associated to
//     "A", regardless of whether this association is defined as being
//     mandatory, or even possible, by the SECA specs.
//
//  2. Convert the SECA Business Models to the equivalent Aruba resources
//     representation:
//
//     In that step, all the SECA resources, including the one passed as
//     parameter and also the other fetched in the previous step, should be
//     converted to equivalent Aruba resource representations.
//
//  3. Resolve Aruba dependencies for referenced objects in the Aruba domain:
//
//     In that step, the plugin should be able to locate and retrieve all the
//     already existing Aruba resources necessary to perform the intended
//     operation, including those that should be mutated, and those that should
//     only be read to retrieve some information.
//
//  4. Mutate the Aruba resources according the received specs:
//
//     In that step, the plugin handler should be able to perform all mutations
//     on the Aruba resources that are necessary to achieve the desired state
//     according to the specs of the SECA resource passed as a parameter.
//
//  5. Trigger the required action to the Aruba Provisioner:
//
//     In that step, the plugin handler should be able to trigger all actions
//     necessary to achieve the wanted state. In practice, it means to handle
//     the Aruba resources in the Kubernetes cluster.
//
//  6. Wait for the results:
//
//     As the plugin calls are intended to be synchronous, in this step the
//     plugin should be able to watch all the affected Aruba resources until
//     they achieve the desired state.
//
//  7. Then return the results:
//
//     In this step, the plugin should return `nil` to indicate "success" or
//     an error to indicate "failure".
package delegated

import (
	"context"

	seca_gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
)

// TODO: this type should be an alias for the Delegator type.
type DelegatedFunc[T seca_gateway_port.IdentifiableResource] func(ctx context.Context, resource T) error

type Delegated[T seca_gateway_port.IdentifiableResource] interface {
	Do(ctx context.Context, resource T) error
}

type GenericDelegated[T seca_gateway_port.IdentifiableResource] struct {
}

var _ Delegated[seca_gateway_port.IdentifiableResource] = (*GenericDelegated[seca_gateway_port.IdentifiableResource])(nil)

func (d *GenericDelegated[T]) Do(ctx context.Context, resource T) error {
	return nil
}
