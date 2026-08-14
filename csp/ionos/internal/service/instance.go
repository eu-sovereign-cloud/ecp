package service

import (
	"context"

	instancectrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/instance"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	instancek8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
)

var _ instancek8s.InstancePlugin = (*Instance)(nil)

type Instance struct {
	Creator    *instancectrl.CreateInstance
	Deleter    *instancectrl.DeleteInstance
	PowerOner  *instancectrl.PowerOnInstance
	PowerOffer *instancectrl.PowerOffInstance
}

func (s *Instance) Update(ctx context.Context, resource *instancedom.Instance) error {
	// TODO implement me
	return s.Creator.Do(ctx, resource)
}

func (s *Instance) Create(ctx context.Context, resource *instancedom.Instance) error {
	return s.Creator.Do(ctx, resource)
}

func (s *Instance) Delete(ctx context.Context, resource *instancedom.Instance) error {
	return s.Deleter.Do(ctx, resource)
}

func (s *Instance) PowerOn(ctx context.Context, resource *instancedom.Instance) error {
	return s.PowerOner.Do(ctx, resource)
}

func (s *Instance) PowerOff(ctx context.Context, resource *instancedom.Instance) error {
	return s.PowerOffer.Do(ctx, resource)
}
