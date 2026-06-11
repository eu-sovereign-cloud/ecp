package service

import (
	"context"

	networkctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/network"
	delegatorplugin "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
)

var _ delegatorplugin.Network = (*Network)(nil)

type Network struct {
	Creator *networkctrl.CreateNetwork
	Deleter *networkctrl.DeleteNetwork
}

func (s *Network) Create(ctx context.Context, resource *regional.NetworkDomain) error {
	return s.Creator.Do(ctx, resource)
}

func (s *Network) Delete(ctx context.Context, resource *regional.NetworkDomain) error {
	return s.Deleter.Do(ctx, resource)
}
