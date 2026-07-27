package handler

import (
	"context"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	computeskudom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	sgdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	sgrdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// instMocks bundles every collaborator of the ComputeInstanceHandler.
type instMocks struct {
	wsRepo      *MockReaderRepo[*wsdom.Workspace]
	nicRepo     *MockReaderRepo[*nicdom.Nic]
	sgRepo      *MockReaderRepo[*sgdom.SecurityGroup]
	sgrRepo     *MockReaderRepo[*sgrdom.SecurityGroupRule]
	skuRepo     *MockReaderRepo[*computeskudom.InstanceSKU]
	prjRepo     *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList]
	subnetRepo  *MockRepository[*v1alpha1.Subnet, *v1alpha1.SubnetList]
	keyPairRepo *MockRepository[*v1alpha1.KeyPair, *v1alpha1.KeyPairList]
	sgArubaRepo *MockRepository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList]
	srArubaRepo *MockRepository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList]
	csRepo      *MockRepository[*v1alpha1.CloudServer, *v1alpha1.CloudServerList]
}

func newInstMocks(ctrl *gomock.Controller) *instMocks {
	return &instMocks{
		wsRepo:      NewMockReaderRepo[*wsdom.Workspace](ctrl),
		nicRepo:     NewMockReaderRepo[*nicdom.Nic](ctrl),
		sgRepo:      NewMockReaderRepo[*sgdom.SecurityGroup](ctrl),
		sgrRepo:     NewMockReaderRepo[*sgrdom.SecurityGroupRule](ctrl),
		skuRepo:     NewMockReaderRepo[*computeskudom.InstanceSKU](ctrl),
		prjRepo:     NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl),
		subnetRepo:  NewMockRepository[*v1alpha1.Subnet, *v1alpha1.SubnetList](ctrl),
		keyPairRepo: NewMockRepository[*v1alpha1.KeyPair, *v1alpha1.KeyPairList](ctrl),
		sgArubaRepo: NewMockRepository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList](ctrl),
		srArubaRepo: NewMockRepository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList](ctrl),
		csRepo:      NewMockRepository[*v1alpha1.CloudServer, *v1alpha1.CloudServerList](ctrl),
	}
}

func (m *instMocks) handler() *ComputeInstanceHandler {
	return NewComputeInstanceHandler(m.wsRepo, m.nicRepo, m.sgRepo, m.sgrRepo, m.skuRepo, m.prjRepo,
		m.subnetRepo, m.keyPairRepo, m.sgArubaRepo, m.srArubaRepo, m.csRepo)
}

func testInstance() *instancedom.Instance {
	return &instancedom.Instance{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: "vm-1"},
			Scope:          res.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
		},
		Spec: instancedom.InstanceSpec{
			PrimaryNicRef: &commondomain.Reference{Resource: "nics/nic-1"},
			BootVolume:    instancedom.VolumeReference{DeviceRef: commondomain.Reference{Resource: "block-storages/boot"}},
			SkuRef:        commondomain.Reference{Resource: "skus/n1.small"},
			SshKeys:       []string{"ssh-rsa AAAA"},
			Zone:          "ITBG-1",
		},
	}
}

func expectProjectActive(m *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList]) {
	m.EXPECT().Load(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, p *v1alpha1.Project) error {
		p.Status.Phase = v1alpha1.ResourcePhaseActive
		return nil
	}).AnyTimes()
}

// expectComputeSku makes the InstanceSKU reader return a SKU of the given capacity; 4 vCPU / 8 GB
// maps to the Aruba flavor CSO4A8.
func expectComputeSku(m *MockReaderRepo[*computeskudom.InstanceSKU], vcpu, ram int) {
	m.EXPECT().Load(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, s **computeskudom.InstanceSKU) error {
		(*s).Spec.VCPU = vcpu
		(*s).Spec.Ram = ram
		return nil
	}).AnyTimes()
}

// expectNic makes the NIC reader return a NIC referencing one subnet and one security group.
func expectNic(m *MockReaderRepo[*nicdom.Nic], subnet, secGroup string) {
	m.EXPECT().Load(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, n **nicdom.Nic) error {
		(*n).Spec.SubnetRef = commondomain.Reference{Resource: "subnets/" + subnet}
		if secGroup != "" {
			(*n).Spec.SecurityGroupRefs = []commondomain.Reference{{Resource: "security-groups/" + secGroup}}
		}
		return nil
	}).AnyTimes()
}

// expectActiveSubnet makes the Aruba subnet list return one active subnet backing the SECA name.
func expectActiveSubnet(m *MockRepository[*v1alpha1.Subnet, *v1alpha1.SubnetList], name, network string) {
	m.EXPECT().List(gomock.Any(), gomock.Any()).Return(&v1alpha1.SubnetList{
		Items: []v1alpha1.Subnet{{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "net-ns", Labels: map[string]string{"seca.subnet/network": network}},
			Spec:       v1alpha1.SubnetSpec{VPCReference: v1alpha1.ResourceReference{Name: network, Namespace: "ws-ns"}},
			Status:     v1alpha1.SubnetStatus{ResourceStatus: v1alpha1.ResourceStatus{Phase: v1alpha1.ResourcePhaseActive}},
		}},
	}, nil).AnyTimes()
}

func TestComputeInstance_create(t *testing.T) {
	tests := []struct {
		name        string
		instance    *instancedom.Instance
		setupMocks  func(*instMocks)
		wantErr     bool
		errContains string
	}{
		{
			name:     "nic not created yet - still processing",
			instance: testInstance(),
			setupMocks: func(m *instMocks) {
				expectWorkspaceActive(m.wsRepo)
				expectProjectActive(m.prjRepo)
				m.nicRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("nic")).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name:     "nic without a subnet - still processing",
			instance: testInstance(),
			setupMocks: func(m *instMocks) {
				expectWorkspaceActive(m.wsRepo)
				expectProjectActive(m.prjRepo)
				m.nicRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes() // NIC present, no subnet ref
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "no ssh key - still processing",
			instance: func() *instancedom.Instance {
				i := testInstance()
				i.Spec.SshKeys = nil
				return i
			}(),
			setupMocks: func(m *instMocks) {
				expectWorkspaceActive(m.wsRepo)
				expectProjectActive(m.prjRepo)
				expectNic(m.nicRepo, "sub-1", "web")
				expectActiveSubnet(m.subnetRepo, "sub-1", "my-network")
				expectComputeSku(m.skuRepo, 4, 8)
				m.sgRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.sgArubaRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.srArubaRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name:     "cloudserver pending - still processing",
			instance: testInstance(),
			setupMocks: func(m *instMocks) {
				expectWorkspaceActive(m.wsRepo)
				expectProjectActive(m.prjRepo)
				expectNic(m.nicRepo, "sub-1", "web")
				expectActiveSubnet(m.subnetRepo, "sub-1", "my-network")
				expectComputeSku(m.skuRepo, 4, 8)
				m.sgRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.sgArubaRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.srArubaRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.keyPairRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.csRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.csRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes() // present but not active
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name:     "success - cloudserver active",
			instance: testInstance(),
			setupMocks: func(m *instMocks) {
				expectWorkspaceActive(m.wsRepo)
				expectProjectActive(m.prjRepo)
				expectNic(m.nicRepo, "sub-1", "web")
				expectActiveSubnet(m.subnetRepo, "sub-1", "my-network")
				expectComputeSku(m.skuRepo, 4, 8)
				// The SECA security group carries one inline rule so a SecurityRule is materialised.
				m.sgRepo.EXPECT().Load(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, sg **sgdom.SecurityGroup) error {
					(*sg).Spec.Rules = []sgdom.SecurityGroupRuleSpec{{Direction: "ingress", Protocol: "tcp", Ports: &sgdom.Ports{From: 22, To: 22}}}
					return nil
				}).AnyTimes()
				m.sgArubaRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.srArubaRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.keyPairRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.csRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.csRepo.EXPECT().Load(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, cs *v1alpha1.CloudServer) error {
					cs.Status.Phase = v1alpha1.ResourcePhaseActive
					return nil
				}).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := newInstMocks(ctrl)
			tt.setupMocks(m)

			err := m.handler().Create(context.Background(), tt.instance)
			assertErr(t, err, tt.wantErr, tt.errContains)
		})
	}
}

func TestComputeInstance_delete(t *testing.T) {
	t.Run("cloudserver gone - deletes key pair", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := newInstMocks(ctrl)

		m.csRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		m.csRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("cs")).AnyTimes()
		m.keyPairRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).Times(1)

		err := m.handler().Delete(context.Background(), testInstance())
		assertErr(t, err, false, "")
	})

	t.Run("cloudserver still present - still processing", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := newInstMocks(ctrl)

		m.csRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		m.csRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes() // still present

		err := m.handler().Delete(context.Background(), testInstance())
		assertErr(t, err, true, "operation still in progress")
	})
}
