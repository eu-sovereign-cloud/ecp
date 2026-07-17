package kubernetes

import (
	"context"

	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

// SecurityGroupPlugin is implemented by CSP plugins that manage SecurityGroup resources.
type SecurityGroupPlugin interface {
	Create(ctx context.Context, resource *securitygroupdom.SecurityGroup) error
	Delete(ctx context.Context, resource *securitygroupdom.SecurityGroup) error
}
