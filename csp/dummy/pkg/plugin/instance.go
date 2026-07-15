package plugin

import (
	"context"
	"log/slog"
	"time"

	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
)

type Instance struct {
	logger *slog.Logger
}

func NewInstance(logger *slog.Logger) *Instance {
	return &Instance{logger: logger}
}

func (i *Instance) Create(ctx context.Context, resource *instancedom.Instance) error {
	return simulateInstance(ctx, "create", resource, instanceDelay(), i.logger)
}

func (i *Instance) Delete(ctx context.Context, resource *instancedom.Instance) error {
	return simulateInstance(ctx, "delete", resource, instanceDelay(), i.logger)
}

func (i *Instance) PowerOn(ctx context.Context, resource *instancedom.Instance) error {
	return simulateInstance(ctx, "power-on", resource, instanceDelay(), i.logger)
}

func (i *Instance) PowerOff(ctx context.Context, resource *instancedom.Instance) error {
	return simulateInstance(ctx, "power-off", resource, instanceDelay(), i.logger)
}

func instanceDelay() time.Duration {
	return networkDelay()
}
