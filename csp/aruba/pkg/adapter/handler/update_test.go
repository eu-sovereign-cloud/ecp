package handler

import (
	"context"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	sgdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"

	adaptconverter "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
)

// A SECA label edit reaches Aruba as a tag change. This is the whole point of the update path:
// tags are the only field Aruba lets an update change on a VPC, so without it a relabelled
// network would diverge from the provider silently.
func TestNetworkHandler_Update_syncsLabelsToTags(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	network := &netdom.Network{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: "net-1"},
			Scope:          res.Scope{Tenant: "acme", Workspace: "ws-1"},
			Labels:         map[string]string{"env": "prod", "team": "platform"},
		},
	}

	m := newNetMocks(ctrl)
	// The real converter, so the test covers the label -> tag mapping end to end.
	handler := NewNetworkHandler(m.wsRepo, m.igwRepo, m.vpcRepo, m.prjRepo, adaptconverter.NewNetworkVPCConverter(), m.wsConv)

	// The live VPC still carries the tags it was created with.
	m.vpcRepo.EXPECT().Load(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, v *v1alpha1.VPC) error {
			v.Spec.Tags = []string{"env-staging"}
			return nil
		}).Times(1)

	var written *v1alpha1.VPC
	m.vpcRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, v *v1alpha1.VPC) error {
			written = v
			return nil
		}).Times(1)

	require.NoError(t, handler.Update(context.Background(), network))
	require.NotNil(t, written, "a changed label must be written through to the VPC")
	require.Equal(t, []string{"env-prod", "team-platform"}, written.Spec.Tags)
}

// Update runs on every reconcile of an active resource. Writing when nothing changed would churn
// the Aruba CR, which the operator watches, on every single pass.
func TestNetworkHandler_Update_writesNothingWhenTagsAlreadyMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	network := &netdom.Network{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: "net-1"},
			Scope:          res.Scope{Tenant: "acme", Workspace: "ws-1"},
			Labels:         map[string]string{"env": "prod"},
		},
	}

	m := newNetMocks(ctrl)
	handler := NewNetworkHandler(m.wsRepo, m.igwRepo, m.vpcRepo, m.prjRepo, adaptconverter.NewNetworkVPCConverter(), m.wsConv)

	m.vpcRepo.EXPECT().Load(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, v *v1alpha1.VPC) error {
			v.Spec.Tags = []string{"env-prod"}
			return nil
		}).Times(1)

	// No Update expectation: the mock controller fails the test if one is issued.
	require.NoError(t, handler.Update(context.Background(), network))
}

// A SECA security group is materialised once per network it is attached in, so relabelling it has
// to reach every materialised copy - and only the ones in its own workspace, for the same reason
// the reap is namespace-scoped.
func TestSecurityGroupHandler_Update_retagsEveryMaterialisedGroup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	domain := &sgdom.SecurityGroup{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: "web"},
			Scope:          res.Scope{Tenant: "acme", Workspace: "ws-1"},
			Labels:         map[string]string{"tier": "frontend"},
		},
	}
	namespace := k8sadapter.ComputeNamespace(domain)

	sgRepo := NewMockRepository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList](ctrl)
	ruleRepo := NewMockRepository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList](ctrl)
	handler := NewSecurityGroupHandler(sgRepo, ruleRepo)

	sgRepo.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, opts ...client.ListOption) (*v1alpha1.SecurityGroupList, error) {
			require.Contains(t, opts, client.InNamespace(namespace), "the retag must stay in its own workspace")
			require.Contains(t, opts, client.MatchingLabels{
				adaptconverter.LabelTenant:        "acme",
				adaptconverter.LabelSecurityGroup: "web",
			})

			return &v1alpha1.SecurityGroupList{Items: []v1alpha1.SecurityGroup{
				{ObjectMeta: metav1.ObjectMeta{Name: "web-prod", Namespace: namespace}, Spec: v1alpha1.SecurityGroupSpec{Tags: []string{"tier-backend"}}},
				{ObjectMeta: metav1.ObjectMeta{Name: "web-dev", Namespace: namespace}, Spec: v1alpha1.SecurityGroupSpec{Tags: []string{"tier-frontend"}}},
			}}, nil
		}).Times(1)

	// Only the group whose tags actually differ is written; the one already in sync is skipped.
	var retagged []string
	sgRepo.EXPECT().Update(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, sg *v1alpha1.SecurityGroup) error {
			retagged = append(retagged, sg.Name)
			require.Equal(t, []string{"tier-frontend"}, sg.Spec.Tags)
			return nil
		}).Times(1)

	require.NoError(t, handler.Update(context.Background(), domain))
	require.Equal(t, []string{"web-prod"}, retagged)
}
