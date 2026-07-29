package plugin

import (
	"context"
	"log/slog"
	"time"

	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

type SecurityGroup struct {
	logger *slog.Logger
}

func NewSecurityGroup(logger *slog.Logger) *SecurityGroup {
	return &SecurityGroup{logger: logger}
}

func (sg *SecurityGroup) Create(ctx context.Context, resource *securitygroupdom.SecurityGroup) error {
	return simulateSecurityGroup(ctx, "create", resource, securityGroupDelay(), sg.logger)
}

func (sg *SecurityGroup) Delete(ctx context.Context, resource *securitygroupdom.SecurityGroup) error {
	return simulateSecurityGroup(ctx, "delete", resource, securityGroupDelay(), sg.logger)
}

func securityGroupDelay() time.Duration {
	return networkDelay()
}
