package delegated

import (
	"context"

	seca_gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	converter_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/converter"
	delegated_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/delegated"
	mutator_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/mutator"
	repository_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/repository"
	resolver_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/resolver"
)

type GenericDelegated[
	S seca_gateway_port.IdentifiableResource, // SECA resource type
	SB any, // SECA bundle type
	AB any, // Aruba bundle type
] struct {
	resolveSECAFunc resolver_port.ResolveDependenciesFunc[S, SB]
	convertFunc     converter_port.ConvertFunc[SB, AB]
	resolvArubaFunc resolver_port.ResolveDependenciesFunc[AB, AB]
	mutateFunc      mutator_port.MutateFunc[AB, SB]
	propagateFunc   repository_port.CLUDFunc[AB]
	conditionFunc   repository_port.WaitConditionFunc[AB]
	watchFunc       repository_port.WaitUntilFunc[AB]
}

var _ delegated_port.Delegated[seca_gateway_port.IdentifiableResource] = (*GenericDelegated[seca_gateway_port.IdentifiableResource, any, any])(nil)

func (d *GenericDelegated[S, SB, AB]) Do(ctx context.Context, resource S) error {
	secaBundle, err := d.resolveSECAFunc(ctx, resource)
	if err != nil {
		return err
	}

	arubaBundle, err := d.convertFunc(secaBundle)
	if err != nil {
		return err
	}

	arubaBundle, err = d.resolvArubaFunc(ctx, arubaBundle)
	if err != nil {
		return err
	}

	if err := d.mutateFunc(arubaBundle, secaBundle); err != nil {
		return err
	}

	if err := d.propagateFunc(ctx, arubaBundle); err != nil {
		return err
	}

	if _, err := d.watchFunc(ctx, arubaBundle, d.conditionFunc); err != nil {
		return err
	}

	return nil
}
