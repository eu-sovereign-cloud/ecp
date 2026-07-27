package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	adaptconverter "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
	genrepo "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/repository"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	sgdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
)

func testSecaSG() *sgdom.SecurityGroup {
	return &sgdom.SecurityGroup{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: "web"},
			Scope:          res.Scope{Tenant: "acme", Workspace: "ws-1"},
		},
	}
}

func sgMocks(ctrl *gomock.Controller) (
	*SecurityGroupHandler,
	*MockRepository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList],
	*MockRepository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList],
) {
	sgRepo := NewMockRepository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList](ctrl)
	srRepo := NewMockRepository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList](ctrl)
	return NewSecurityGroupHandler(sgRepo, srRepo), sgRepo, srRepo
}

// Create must never touch the Aruba side: materialisation is the instance handler's job. Any
// repository call here would be an unexpected call and fail the mock controller.
func TestSecurityGroup_create_isNoop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	h, _, _ := sgMocks(ctrl)

	assertErr(t, h.Create(context.Background(), testSecaSG()), false, "")
}

func TestSecurityGroup_delete(t *testing.T) {
	group := v1alpha1.SecurityGroup{ObjectMeta: metav1.ObjectMeta{Name: "web-net1", Namespace: "ws-ns"}}
	rule := func(n string) v1alpha1.SecurityRule {
		return v1alpha1.SecurityRule{ObjectMeta: metav1.ObjectMeta{Name: n, Namespace: "ws-ns"}}
	}

	t.Run("no materialised groups - nothing to delete", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		h, sgRepo, _ := sgMocks(ctrl)

		sgRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.SecurityGroupList{}, nil).Times(1)
		// No rule List and no Delete expectations: any such call fails the controller.

		assertErr(t, h.Delete(context.Background(), testSecaSG()), false, "")
	})

	t.Run("group with rules - reaps every rule then the group", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		h, sgRepo, srRepo := sgMocks(ctrl)

		sgRepo.EXPECT().List(gomock.Any(), gomock.Any()).
			Return(&v1alpha1.SecurityGroupList{Items: []v1alpha1.SecurityGroup{group}}, nil).Times(1)
		srRepo.EXPECT().List(gomock.Any(), gomock.Any()).
			Return(&v1alpha1.SecurityRuleList{Items: []v1alpha1.SecurityRule{rule("web-net1-r0"), rule("web-net1-r1")}}, nil).Times(1)
		srRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(2)
		sgRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		assertErr(t, h.Delete(context.Background(), testSecaSG()), false, "")
	})

	t.Run("NotFound on delete is ignored", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		h, sgRepo, srRepo := sgMocks(ctrl)

		sgRepo.EXPECT().List(gomock.Any(), gomock.Any()).
			Return(&v1alpha1.SecurityGroupList{Items: []v1alpha1.SecurityGroup{group}}, nil).Times(1)
		srRepo.EXPECT().List(gomock.Any(), gomock.Any()).
			Return(&v1alpha1.SecurityRuleList{Items: []v1alpha1.SecurityRule{rule("web-net1-r0")}}, nil).Times(1)
		srRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(notFoundErr("web-net1-r0")).Times(1)
		sgRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(notFoundErr("web-net1")).Times(1)

		assertErr(t, h.Delete(context.Background(), testSecaSG()), false, "")
	})

	t.Run("group list error propagates", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		h, sgRepo, _ := sgMocks(ctrl)

		sgRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, errors.New("boom")).Times(1)

		assertErr(t, h.Delete(context.Background(), testSecaSG()), true, "boom")
	})

	t.Run("rule delete error bails before deleting the group", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		h, sgRepo, srRepo := sgMocks(ctrl)

		sgRepo.EXPECT().List(gomock.Any(), gomock.Any()).
			Return(&v1alpha1.SecurityGroupList{Items: []v1alpha1.SecurityGroup{group}}, nil).Times(1)
		srRepo.EXPECT().List(gomock.Any(), gomock.Any()).
			Return(&v1alpha1.SecurityRuleList{Items: []v1alpha1.SecurityRule{rule("web-net1-r0")}}, nil).Times(1)
		srRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(errors.New("boom")).Times(1)
		// The group must NOT be deleted while a rule delete is failing: no sgRepo.Delete expectation.

		assertErr(t, h.Delete(context.Background(), testSecaSG()), true, "boom")
	})
}

// The reap runs through GenericRepository, which is a plain cluster-wide client.List: the ECP's
// per-workspace namespaces only scope it if the list options say so. Two workspaces of one tenant
// can each hold a SECA group "web" attached in a network named "prod", and both materialise as
// SecurityGroup "web-prod" - same name, different namespace. Deleting one must leave the other's
// firewall intact. Backed by a fake client rather than a mock so the filtering actually happens.
func TestSecurityGroup_delete_staysInsideItsWorkspace(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	nsFor := func(workspace string) string {
		return k8sadapter.ComputeNamespace(&res.Scope{Tenant: "acme", Workspace: workspace})
	}
	vpcRef := v1alpha1.ResourceReference{Name: "prod"}
	prjRef := v1alpha1.ResourceReference{Name: "project"}
	rules := []adaptconverter.RuleSpec{{Direction: "ingress", Protocol: "tcp", PortFrom: 22, PortTo: 22}}

	var objs []client.Object
	for _, ws := range []string{"ws-1", "ws-2"} {
		sg := adaptconverter.BuildSecurityGroup("web", "prod", "", "acme", nsFor(ws), vpcRef, prjRef)
		objs = append(objs, sg)
		for _, rule := range adaptconverter.BuildSecurityRules(rules, sg.Name, "", "acme", nsFor(ws), vpcRef, prjRef) {
			objs = append(objs, rule)
		}
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
	h := NewSecurityGroupHandler(
		genrepo.NewGenericRepository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList](ctx, fakeClient, nil),
		genrepo.NewGenericRepository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList](ctx, fakeClient, nil),
	)

	require.NoError(t, h.Delete(ctx, testSecaSG())) // testSecaSG is acme/ws-1

	sgKey := func(ws string) client.ObjectKey {
		return client.ObjectKey{Name: "web-prod", Namespace: nsFor(ws)}
	}
	ruleCount := func(ws string) int {
		list := &v1alpha1.SecurityRuleList{}
		require.NoError(t, fakeClient.List(ctx, list, client.InNamespace(nsFor(ws))))
		return len(list.Items)
	}

	// ws-1 is the workspace being reaped: its group and rules must be gone.
	require.True(t, apierrors.IsNotFound(fakeClient.Get(ctx, sgKey("ws-1"), &v1alpha1.SecurityGroup{})))
	require.Zero(t, ruleCount("ws-1"))

	// ws-2 is a bystander: it must be untouched.
	require.NoError(t, fakeClient.Get(ctx, sgKey("ws-2"), &v1alpha1.SecurityGroup{}))
	require.Equal(t, 1, ruleCount("ws-2"))
}
