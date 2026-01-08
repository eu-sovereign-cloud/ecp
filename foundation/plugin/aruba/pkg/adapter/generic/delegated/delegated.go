package delegated

import (
	"context"
	"errors"

	seca_gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	converter_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/converter"
	delegated_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/delegated"
	mutator_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/mutator"
	resolver_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/resolver"
)

type GenericDelegated[
	S seca_gateway_port.IdentifiableResource, // SECA resource type
	SB any, // SECA bundle type
	AB any, // Aruba bundle type
] struct {
	secaResolver  resolver_port.DependenciesResolver[S, SB]
	converter     converter_port.Converter[SB, AB]
	arubaResolver resolver_port.DependenciesResolver[AB, AB]
	mutator       mutator_port.Mutator[AB, SB]
}

var _ delegated_port.Delegated[seca_gateway_port.IdentifiableResource] = (*GenericDelegated[seca_gateway_port.IdentifiableResource, any, any])(nil)

func (d *GenericDelegated[S, SB, AB]) Do(ctx context.Context, resource S) error {
	secaBundle, err := d.secaResolver.ResolveDependencies(ctx, resource)
	if err != nil {
		return err
	}

	arubaBundle, err := d.converter.FromSECAToAruba(secaBundle)
	if err != nil {
		return err
	}

	arubaBundle, err = d.arubaResolver.ResolveDependencies(ctx, arubaBundle)
	if err != nil {
		return err
	}

	if err := d.mutator.Mutate(arubaBundle, secaBundle); err != nil {
		return err
	}

	return errors.New("not implemented")
}
