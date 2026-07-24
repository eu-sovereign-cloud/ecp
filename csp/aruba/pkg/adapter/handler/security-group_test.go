package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
