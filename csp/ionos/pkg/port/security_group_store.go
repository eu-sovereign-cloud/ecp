package port

import (
	"context"

	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

// SecurityGroupStore reconciles a SECA security group against an IONOS Network Security Group
// (NSG) and the firewall rules that belong to it.
type SecurityGroupStore interface {
	Create(ctx context.Context, domain *securitygroupdom.SecurityGroup) error
	Delete(ctx context.Context, domain *securitygroupdom.SecurityGroup) error
}
