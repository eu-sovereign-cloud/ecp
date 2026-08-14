package service

import (
	"context"

	subnetctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/subnet"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
	subnetk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
)

var _ subnetk8s.SubnetPlugin = (*Subnet)(nil)

type Subnet struct {
	Creator *subnetctrl.CreateSubnet
	Deleter *subnetctrl.DeleteSubnet
}

func (s *Subnet) Update(ctx context.Context, resource *subnetdom.Subnet) error {
	// TODO implement me
	panic("implement me")
}

func (s *Subnet) Create(ctx context.Context, resource *subnetdom.Subnet) error {
	return s.Creator.Do(ctx, resource)
}

func (s *Subnet) Delete(ctx context.Context, resource *subnetdom.Subnet) error {
	return s.Deleter.Do(ctx, resource)
}
