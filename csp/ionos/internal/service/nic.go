package service

import (
	"context"

	nicctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/nic"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
)

var _ nick8s.NicPlugin = (*Nic)(nil)

type Nic struct {
	Creator *nicctrl.CreateNic
	Deleter *nicctrl.DeleteNic
}

func (s *Nic) Create(ctx context.Context, resource *nicdom.Nic) error {
	return s.Creator.Do(ctx, resource)
}

func (s *Nic) Delete(ctx context.Context, resource *nicdom.Nic) error {
	return s.Deleter.Do(ctx, resource)
}
