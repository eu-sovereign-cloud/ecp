package service

import (
	"context"

	netdom "github.com/eu-sovereign-cloud/ecp/resources/regional/network/networks/v1/domain"
	netk8s "github.com/eu-sovereign-cloud/ecp/resources/regional/network/networks/v1/backend/kubernetes"
	networkctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/network"
)

var _ netk8s.NetworkPlugin = (*Network)(nil)

type Network struct {
	Creator *networkctrl.CreateNetwork
	Deleter *networkctrl.DeleteNetwork
}

func (s *Network) Create(ctx context.Context, resource *netdom.NetworkDomain) error {
	return s.Creator.Do(ctx, resource)
}

func (s *Network) Delete(ctx context.Context, resource *netdom.NetworkDomain) error {
	return s.Deleter.Do(ctx, resource)
}
