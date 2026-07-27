package handler

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	sgdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	sgk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"

	adaptconverter "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/repository"
)

var _ sgk8s.SecurityGroupPlugin = (*SecurityGroupHandler)(nil)

// SecurityGroupHandler owns the lifecycle of a SECA security group on the Aruba side. Create is a
// no-op - a SECA group carries no VPC, so the Aruba SecurityGroup is materialised per VPC by the
// compute-instance handler at attach time (see instance.go). Delete reaps those materialised
// resources: the instance handler deliberately leaves them in place on instance deletion so they
// can be shared, which means the SECA group is the only thing that can clean them up.
type SecurityGroupHandler struct {
	secGroupRepository repository.Repository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList]
	secRuleRepository  repository.Repository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList]
}

// NewSecurityGroupHandler wires the Aruba SecurityGroup and SecurityRule repositories the handler
// reaps through on delete.
func NewSecurityGroupHandler(
	secGroupRepo repository.Repository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList],
	secRuleRepo repository.Repository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList],
) *SecurityGroupHandler {
	return &SecurityGroupHandler{secGroupRepository: secGroupRepo, secRuleRepository: secRuleRepo}
}

// Create is a no-op: there is nothing to materialise until an instance binds this group to a VPC.
func (h *SecurityGroupHandler) Create(_ context.Context, _ *sgdom.SecurityGroup) error { return nil }

// Delete reaps every Aruba SecurityGroup (and its SecurityRules) materialised for this SECA group.
// One SECA group is materialised once per VPC it was attached in, so all labelled matches are
// removed. Idempotent: a missing object is treated as already reaped, so a retried delete after a
// partial failure is harmless.
//
// Every list here is namespace-scoped. The repository issues a plain cluster-wide client.List, so
// labels alone do not confine the reap to this SECA group's own workspace - and the materialised
// name (<seca>-<network>) is only unique within one, since a sibling workspace of the same tenant
// may hold its own group of the same name in a network of the same name.
func (h *SecurityGroupHandler) Delete(ctx context.Context, domain *sgdom.SecurityGroup) error {
	// The instance handler materialises these into the workspace namespace (see instance.go).
	workspaceNamespace := client.InNamespace(k8sadapter.ComputeNamespace(domain))

	groups, err := h.secGroupRepository.List(ctx, workspaceNamespace, client.MatchingLabels{
		adaptconverter.LabelTenant:        domain.GetTenant(),
		adaptconverter.LabelSecurityGroup: domain.Name,
	})
	if err != nil {
		return err
	}

	for i := range groups.Items {
		sg := &groups.Items[i]
		// A rule is labelled with the materialised group name, not the SECA name (see
		// converter.BuildSecurityRules), so list per materialised group - in that group's own
		// namespace, which is where BuildSecurityRules puts them.
		rules, err := h.secRuleRepository.List(ctx, client.InNamespace(sg.Namespace), client.MatchingLabels{
			adaptconverter.LabelSecurityGroup: sg.Name,
		})
		if err != nil {
			return err
		}
		for j := range rules.Items {
			if err := h.secRuleRepository.Delete(ctx, &rules.Items[j]); err != nil && !apierrors.IsNotFound(err) {
				return err
			}
		}
		if err := h.secGroupRepository.Delete(ctx, sg); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
