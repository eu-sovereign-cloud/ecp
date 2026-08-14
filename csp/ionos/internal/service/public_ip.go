package service

import (
	"context"

	publicipctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/public_ip"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
	publicipk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip/backend/kubernetes"
)

var _ publicipk8s.PublicIpPlugin = (*PublicIP)(nil)

type PublicIP struct {
	Creator *publicipctrl.CreatePublicIP
	Deleter *publicipctrl.DeletePublicIP
}

func (s *PublicIP) Update(ctx context.Context, resource *publicipdom.PublicIp) error {
	// TODO implement me
	panic("implement me")
}

func (s *PublicIP) Create(ctx context.Context, resource *publicipdom.PublicIp) error {
	return s.Creator.Do(ctx, resource)
}

func (s *PublicIP) Delete(ctx context.Context, resource *publicipdom.PublicIp) error {
	return s.Deleter.Do(ctx, resource)
}
