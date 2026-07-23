package handler

import (
	"context"
	"fmt"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"go.uber.org/mock/gomock"

	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// subnetMocks bundles every collaborator of the SubnetHandler so each test case
// can wire only the behaviour it needs.
type subnetMocks struct {
	wsRepo     *MockReaderRepo[*wsdom.Workspace]
	subnetRepo *MockRepository[*v1alpha1.Subnet, *v1alpha1.SubnetList]
	vpcRepo    *MockRepository[*v1alpha1.VPC, *v1alpha1.VPCList]
	prjRepo    *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList]
	subnetConv *MockConverter[*subnetdom.Subnet, *v1alpha1.Subnet]
	wsConv     *MockConverter[*wsdom.Workspace, *v1alpha1.Project]
}

func newSubnetMocks(ctrl *gomock.Controller) *subnetMocks {
	return &subnetMocks{
		wsRepo:     NewMockReaderRepo[*wsdom.Workspace](ctrl),
		subnetRepo: NewMockRepository[*v1alpha1.Subnet, *v1alpha1.SubnetList](ctrl),
		vpcRepo:    NewMockRepository[*v1alpha1.VPC, *v1alpha1.VPCList](ctrl),
		prjRepo:    NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl),
		subnetConv: NewMockConverter[*subnetdom.Subnet, *v1alpha1.Subnet](ctrl),
		wsConv:     NewMockConverter[*wsdom.Workspace, *v1alpha1.Project](ctrl),
	}
}

func (m *subnetMocks) handler() *SubnetHandler {
	return NewSubnetHandler(m.wsRepo, m.subnetRepo, m.vpcRepo, m.prjRepo, m.subnetConv, m.wsConv)
}

// arubaSubnet returns a converted Aruba Subnet in the given phase, referencing the VPC the
// SubnetHandler probes for.
func arubaSubnet(phase v1alpha1.ResourcePhase) *v1alpha1.Subnet {
	return &v1alpha1.Subnet{
		Spec: v1alpha1.SubnetSpec{
			CIDR:         "10.0.1.0/24",
			VPCReference: v1alpha1.ResourceReference{Name: "my-network", Namespace: "network-ns"},
		},
		Status: v1alpha1.SubnetStatus{
			ResourceStatus: v1alpha1.ResourceStatus{Phase: phase},
		},
	}
}

// expectVpcPhase makes the VPC repository report a VPC in the given phase, which the subnet
// create flow requires to be active before it proceeds.
func expectVpcPhase(m *MockRepository[*v1alpha1.VPC, *v1alpha1.VPCList], phase v1alpha1.ResourcePhase) {
	m.EXPECT().
		Load(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, v *v1alpha1.VPC) error {
			v.Status.Phase = phase
			return nil
		}).
		AnyTimes()
}

func TestSubnet_create(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*subnetMocks)
		wantErr     bool
		errContains string
	}{
		{
			name: "workspace not ready - still processing",
			setupMocks: func(m *subnetMocks) {
				m.wsRepo.EXPECT().
					Load(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("workspace not found")).
					AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "conversion error",
			setupMocks: func(m *subnetMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.subnetConv.EXPECT().
					FromSECAToAruba(gomock.Any()).
					Return(nil, fmt.Errorf("conversion error"))
			},
			wantErr:     true,
			errContains: "conversion error",
		},
		{
			name: "project not ready - still processing",
			setupMocks: func(m *subnetMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.subnetConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(arubaSubnet(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("project")).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			// Aruba rejects a subnet whose VPC does not exist yet, so the flow
			// waits for the network handler to create it.
			name: "vpc missing - still processing",
			setupMocks: func(m *subnetMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.subnetConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(arubaSubnet(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.vpcRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("vpc")).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "vpc not active - still processing",
			setupMocks: func(m *subnetMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.subnetConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(arubaSubnet(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				expectVpcPhase(m.vpcRepo, v1alpha1.ResourcePhaseCreating)
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "create error",
			setupMocks: func(m *subnetMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.subnetConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(arubaSubnet(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				expectVpcPhase(m.vpcRepo, v1alpha1.ResourcePhaseActive)
				m.subnetRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("subnet")).AnyTimes()
				m.subnetRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(fmt.Errorf("creation error"))
			},
			wantErr:     true,
			errContains: "creation error",
		},
		{
			name: "pending creation - still processing",
			setupMocks: func(m *subnetMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.subnetConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(arubaSubnet(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				expectVpcPhase(m.vpcRepo, v1alpha1.ResourcePhaseActive)
				// Present but not active yet, so the check reports "not done".
				m.subnetRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.subnetRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "create idempotent on already exists - still processing",
			setupMocks: func(m *subnetMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.subnetConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(arubaSubnet(v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				expectVpcPhase(m.vpcRepo, v1alpha1.ResourcePhaseActive)
				m.subnetRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.subnetRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(alreadyExistsErr("subnet")).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "success create",
			setupMocks: func(m *subnetMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.subnetConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(arubaSubnet(v1alpha1.ResourcePhaseActive), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				expectVpcPhase(m.vpcRepo, v1alpha1.ResourcePhaseActive)
				// Present and active, so the check reports "done".
				m.subnetRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.subnetRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := newSubnetMocks(ctrl)
			tt.setupMocks(m)

			err := m.handler().Create(context.Background(), &subnetdom.Subnet{
				Spec: subnetdom.SubnetSpec{Cidr: subnetdom.CIDR{IPv4: "10.0.1.0/24"}},
			})

			assertErr(t, err, tt.wantErr, tt.errContains)
		})
	}
}

func TestSubnet_delete(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*subnetMocks)
		wantErr     bool
		errContains string
	}{
		{
			name: "conversion error",
			setupMocks: func(m *subnetMocks) {
				m.subnetConv.EXPECT().
					FromSECAToAruba(gomock.Any()).
					Return(nil, fmt.Errorf("conversion error"))
			},
			wantErr:     true,
			errContains: "conversion error",
		},
		{
			name: "delete error",
			setupMocks: func(m *subnetMocks) {
				m.subnetConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(arubaSubnet(v1alpha1.ResourcePhaseDeleting), nil).AnyTimes()
				m.subnetRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.subnetRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(fmt.Errorf("deletion error"))
			},
			wantErr:     true,
			errContains: "deletion error",
		},
		{
			name: "pending deletion - still processing",
			setupMocks: func(m *subnetMocks) {
				m.subnetConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(arubaSubnet(v1alpha1.ResourcePhaseDeleting), nil).AnyTimes()
				m.subnetRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.subnetRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "success delete",
			setupMocks: func(m *subnetMocks) {
				m.subnetConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(arubaSubnet(v1alpha1.ResourcePhaseDeleted), nil).AnyTimes()
				// Gone, so the check reports "done".
				m.subnetRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("subnet")).AnyTimes()
				m.subnetRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := newSubnetMocks(ctrl)
			tt.setupMocks(m)

			err := m.handler().Delete(context.Background(), &subnetdom.Subnet{})

			assertErr(t, err, tt.wantErr, tt.errContains)
		})
	}
}
