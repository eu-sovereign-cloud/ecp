package handler

import (
	"context"
	"fmt"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"go.uber.org/mock/gomock"

	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	igwdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// netMocks bundles every collaborator of the NetworkHandler so each test case
// can wire only the behaviour it needs.
type netMocks struct {
	wsRepo  *MockReaderRepo[*wsdom.Workspace]
	igwRepo *MockReaderRepo[*igwdom.InternetGateway]
	vpcRepo *MockRepository[*v1alpha1.VPC, *v1alpha1.VPCList]
	prjRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList]
	netConv *MockConverter[*netdom.Network, *v1alpha1.VPC]
	wsConv  *MockConverter[*wsdom.Workspace, *v1alpha1.Project]
}

func newNetMocks(ctrl *gomock.Controller) *netMocks {
	return &netMocks{
		wsRepo:  NewMockReaderRepo[*wsdom.Workspace](ctrl),
		igwRepo: NewMockReaderRepo[*igwdom.InternetGateway](ctrl),
		vpcRepo: NewMockRepository[*v1alpha1.VPC, *v1alpha1.VPCList](ctrl),
		prjRepo: NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl),
		netConv: NewMockConverter[*netdom.Network, *v1alpha1.VPC](ctrl),
		wsConv:  NewMockConverter[*wsdom.Workspace, *v1alpha1.Project](ctrl),
	}
}

func (m *netMocks) handler() *NetworkHandler {
	return NewNetworkHandler(m.wsRepo, m.igwRepo, m.vpcRepo, m.prjRepo, m.netConv, m.wsConv)
}

// vpc returns a converted Aruba VPC in the given phase.
func vpc(phase v1alpha1.ResourcePhase) *v1alpha1.VPC {
	return &v1alpha1.VPC{
		Status: v1alpha1.VPCStatus{
			ResourceStatus: v1alpha1.ResourceStatus{Phase: phase},
		},
	}
}

// expectInternetGateway makes the internet gateway reader report count gateways in the
// workspace, which the network create flow requires before it proceeds.
func expectInternetGateway(m *MockReaderRepo[*igwdom.InternetGateway], count int) {
	m.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ res.ListFilter, list *[]*igwdom.InternetGateway) (*string, error) {
			for range count {
				*list = append(*list, &igwdom.InternetGateway{})
			}
			return nil, nil
		}).
		AnyTimes()
}

func TestNetwork_create(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*netMocks)
		wantErr     bool
		errContains string
	}{
		{
			name: "workspace not ready - still processing",
			setupMocks: func(m *netMocks) {
				m.wsRepo.EXPECT().
					Load(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("workspace not found")).
					AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			// An Aruba VPC always provides internet egress, so the SECA gateway
			// representing it must exist before the VPC is created.
			name: "no internet gateway - still processing",
			setupMocks: func(m *netMocks) {
				expectWorkspaceActive(m.wsRepo)
				expectInternetGateway(m.igwRepo, 0)
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "internet gateway list error",
			setupMocks: func(m *netMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.igwRepo.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, fmt.Errorf("list error")).
					AnyTimes()
			},
			wantErr:     true,
			errContains: "list error",
		},
		{
			name: "conversion error",
			setupMocks: func(m *netMocks) {
				expectWorkspaceActive(m.wsRepo)
				expectInternetGateway(m.igwRepo, 1)
				m.wsConv.EXPECT().
					FromSECAToAruba(gomock.Any()).
					Return(nil, fmt.Errorf("conversion error"))
			},
			wantErr:     true,
			errContains: "conversion error",
		},
		{
			name: "project not ready - still processing",
			setupMocks: func(m *netMocks) {
				expectWorkspaceActive(m.wsRepo)
				expectInternetGateway(m.igwRepo, 1)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.netConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(vpc(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("project")).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "create error",
			setupMocks: func(m *netMocks) {
				expectWorkspaceActive(m.wsRepo)
				expectInternetGateway(m.igwRepo, 1)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.netConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(vpc(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.vpcRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("vpc")).AnyTimes()
				m.vpcRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(fmt.Errorf("creation error"))
			},
			wantErr:     true,
			errContains: "creation error",
		},
		{
			name: "pending creation - still processing",
			setupMocks: func(m *netMocks) {
				expectWorkspaceActive(m.wsRepo)
				expectInternetGateway(m.igwRepo, 1)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.netConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(vpc(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				// Present but not active yet, so the check reports "not done".
				m.vpcRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.vpcRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "create idempotent on already exists - still processing",
			setupMocks: func(m *netMocks) {
				expectWorkspaceActive(m.wsRepo)
				expectInternetGateway(m.igwRepo, 1)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.netConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(vpc(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.vpcRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.vpcRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(alreadyExistsErr("vpc")).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "success create",
			setupMocks: func(m *netMocks) {
				expectWorkspaceActive(m.wsRepo)
				expectInternetGateway(m.igwRepo, 1)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.netConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(vpc(v1alpha1.ResourcePhaseActive), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				// Present and active, so the check reports "done".
				m.vpcRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.vpcRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := newNetMocks(ctrl)
			tt.setupMocks(m)

			err := m.handler().Create(context.Background(), &netdom.Network{
				Spec: netdom.NetworkSpec{CIDR: netdom.CIDR{IPv4: "10.0.0.0/16"}},
			})

			assertErr(t, err, tt.wantErr, tt.errContains)
		})
	}
}

func TestNetwork_delete(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*netMocks)
		wantErr     bool
		errContains string
	}{
		{
			name: "conversion error",
			setupMocks: func(m *netMocks) {
				m.netConv.EXPECT().
					FromSECAToAruba(gomock.Any()).
					Return(nil, fmt.Errorf("conversion error"))
			},
			wantErr:     true,
			errContains: "conversion error",
		},
		{
			name: "delete error",
			setupMocks: func(m *netMocks) {
				m.netConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(vpc(v1alpha1.ResourcePhaseDeleting), nil).AnyTimes()
				m.vpcRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.vpcRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(fmt.Errorf("deletion error"))
			},
			wantErr:     true,
			errContains: "deletion error",
		},
		{
			name: "pending deletion - still processing",
			setupMocks: func(m *netMocks) {
				m.netConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(vpc(v1alpha1.ResourcePhaseDeleting), nil).AnyTimes()
				m.vpcRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.vpcRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "success delete",
			setupMocks: func(m *netMocks) {
				m.netConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(vpc(v1alpha1.ResourcePhaseDeleted), nil).AnyTimes()
				// Gone, so the check reports "done".
				m.vpcRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("vpc")).AnyTimes()
				m.vpcRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := newNetMocks(ctrl)
			tt.setupMocks(m)

			err := m.handler().Delete(context.Background(), &netdom.Network{})

			assertErr(t, err, tt.wantErr, tt.errContains)
		})
	}
}
