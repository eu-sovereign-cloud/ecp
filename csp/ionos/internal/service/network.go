package service

import (
	"context"

	networkctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/network"
	netdom "github.com/eu-sovereign-cloud/ecp/resources/network/networks/v1"
	netk8s "github.com/eu-sovereign-cloud/ecp/resources/network/networks/v1/backend/kubernetes"
)

var _ netk8s.NetworkPlugin = (*Network)(nil)

type Network struct {
	Creator *networkctrl.CreateNetwork
	Deleter *networkctrl.DeleteNetwork
}

func (s *Network) Create(ctx context.Context, resource *netdom.Network) error {
	return s.Creator.Do(ctx, resource)
}

func (s *Network) Delete(ctx context.Context, resource *netdom.Network) error {
	return s.Deleter.Do(ctx, resource)
}
