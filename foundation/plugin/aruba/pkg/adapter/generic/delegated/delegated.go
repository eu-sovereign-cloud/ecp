package delegated

import (
	"context"
	"errors"

	seca_gateway_port "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	converter_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/converter"
	delegated_port "github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/delegated"
)

type GenericDelegated[S seca_gateway_port.IdentifiableResource, A any] struct {
	converter converter_port.Converter[S, A]
}

var _ delegated_port.Delegated[seca_gateway_port.IdentifiableResource] = (*GenericDelegated[seca_gateway_port.IdentifiableResource, any])(nil)

func (d *GenericDelegated[S, A]) Do(ctx context.Context, resource S) error {
	_, err := d.converter.FromSECAToAruba(resource)
	if err != nil {
		return err
	}

	return errors.New("not implemented")
}
