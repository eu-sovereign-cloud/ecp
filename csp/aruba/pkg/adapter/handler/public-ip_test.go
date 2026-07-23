package handler

import (
	"context"
	"fmt"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"go.uber.org/mock/gomock"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// pipMocks bundles every collaborator of the PublicIpHandler so each test case
// can wire only the behaviour it needs.
type pipMocks struct {
	wsRepo  *MockReaderRepo[*wsdom.Workspace]
	eipRepo *MockRepository[*v1alpha1.ElasticIP, *v1alpha1.ElasticIPList]
	prjRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList]
	pipConv *MockConverter[*publicipdom.PublicIp, *v1alpha1.ElasticIP]
	wsConv  *MockConverter[*wsdom.Workspace, *v1alpha1.Project]
}

func newPipMocks(ctrl *gomock.Controller) *pipMocks {
	return &pipMocks{
		wsRepo:  NewMockReaderRepo[*wsdom.Workspace](ctrl),
		eipRepo: NewMockRepository[*v1alpha1.ElasticIP, *v1alpha1.ElasticIPList](ctrl),
		prjRepo: NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl),
		pipConv: NewMockConverter[*publicipdom.PublicIp, *v1alpha1.ElasticIP](ctrl),
		wsConv:  NewMockConverter[*wsdom.Workspace, *v1alpha1.Project](ctrl),
	}
}

func (m *pipMocks) handler() *PublicIpHandler {
	return NewPublicIpHandler(m.wsRepo, m.eipRepo, m.prjRepo, m.pipConv, m.wsConv)
}

// elasticIP returns a converted Aruba ElasticIP in the given phase.
func elasticIP(phase v1alpha1.ResourcePhase) *v1alpha1.ElasticIP {
	return &v1alpha1.ElasticIP{
		Status: v1alpha1.ElasticIPStatus{
			ResourceStatus: v1alpha1.ResourceStatus{Phase: phase},
		},
	}
}

func TestPublicIp_create(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*pipMocks)
		wantErr     bool
		errContains string
	}{
		{
			name: "workspace not ready - still processing",
			setupMocks: func(m *pipMocks) {
				m.wsRepo.EXPECT().
					Load(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("workspace not found")).
					AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			// The converter rejects specs Aruba cannot honour (BYOIP, IPv6); that
			// is a real error, not a wait, so it must surface as one.
			name: "unsupported spec - conversion error",
			setupMocks: func(m *pipMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.pipConv.EXPECT().
					FromSECAToAruba(gomock.Any()).
					Return(nil, fmt.Errorf("bring-your-own-IP not supported"))
			},
			wantErr:     true,
			errContains: "bring-your-own-IP",
		},
		{
			name: "project not ready - still processing",
			setupMocks: func(m *pipMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.pipConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(elasticIP(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("project")).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "create error",
			setupMocks: func(m *pipMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.pipConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(elasticIP(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.eipRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("elastic-ip")).AnyTimes()
				m.eipRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(fmt.Errorf("creation error"))
			},
			wantErr:     true,
			errContains: "creation error",
		},
		{
			name: "pending creation - still processing",
			setupMocks: func(m *pipMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.pipConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(elasticIP(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				// Present but not active yet, so the check reports "not done".
				m.eipRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.eipRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "create idempotent on already exists - still processing",
			setupMocks: func(m *pipMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.pipConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(elasticIP(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.eipRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.eipRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(alreadyExistsErr("elastic-ip")).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "success create",
			setupMocks: func(m *pipMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.pipConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(elasticIP(v1alpha1.ResourcePhaseActive), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				// Present and active, so the check reports "done".
				m.eipRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.eipRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := newPipMocks(ctrl)
			tt.setupMocks(m)

			err := m.handler().Create(context.Background(), &publicipdom.PublicIp{
				Spec: publicipdom.PublicIpSpec{Version: commondomain.IPVersionIPv4},
			})

			assertErr(t, err, tt.wantErr, tt.errContains)
		})
	}
}

func TestPublicIp_delete(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*pipMocks)
		wantErr     bool
		errContains string
	}{
		{
			name: "conversion error",
			setupMocks: func(m *pipMocks) {
				m.pipConv.EXPECT().
					FromSECAToAruba(gomock.Any()).
					Return(nil, fmt.Errorf("conversion error"))
			},
			wantErr:     true,
			errContains: "conversion error",
		},
		{
			name: "delete error",
			setupMocks: func(m *pipMocks) {
				m.pipConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(elasticIP(v1alpha1.ResourcePhaseDeleting), nil).AnyTimes()
				m.eipRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.eipRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(fmt.Errorf("deletion error"))
			},
			wantErr:     true,
			errContains: "deletion error",
		},
		{
			name: "pending deletion - still processing",
			setupMocks: func(m *pipMocks) {
				m.pipConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(elasticIP(v1alpha1.ResourcePhaseDeleting), nil).AnyTimes()
				m.eipRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.eipRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "success delete",
			setupMocks: func(m *pipMocks) {
				m.pipConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(elasticIP(v1alpha1.ResourcePhaseDeleted), nil).AnyTimes()
				// Gone, so the check reports "done".
				m.eipRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("elastic-ip")).AnyTimes()
				m.eipRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := newPipMocks(ctrl)
			tt.setupMocks(m)

			err := m.handler().Delete(context.Background(), &publicipdom.PublicIp{})

			assertErr(t, err, tt.wantErr, tt.errContains)
		})
	}
}
