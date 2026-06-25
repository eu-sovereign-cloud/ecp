package handler

import (
	"context"
	"fmt"
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	refdom "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	bsdomblock "github.com/eu-sovereign-cloud/ecp/resource/storage/block-storage/v1"
	bsdomsku "github.com/eu-sovereign-cloud/ecp/resource/storage/storage-sku/v1"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// bsMocks bundles every collaborator of the BlockStorageHandler so each test
// case can wire only the behaviour it needs.
type bsMocks struct {
	wsRepo  *MockReaderRepo[*wsdom.Workspace]
	skuRepo *MockReaderRepo[*bsdomsku.StorageSKU]
	bsRepo  *MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList]
	prjRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList]
	bsConv  *MockConverter[*bsdomblock.BlockStorage, *v1alpha1.BlockStorage]
	wsConv  *MockConverter[*wsdom.Workspace, *v1alpha1.Project]
}

func newBsMocks(ctrl *gomock.Controller) *bsMocks {
	return &bsMocks{
		wsRepo:  NewMockReaderRepo[*wsdom.Workspace](ctrl),
		skuRepo: NewMockReaderRepo[*bsdomsku.StorageSKU](ctrl),
		bsRepo:  NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl),
		prjRepo: NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl),
		bsConv:  NewMockConverter[*bsdomblock.BlockStorage, *v1alpha1.BlockStorage](ctrl),
		wsConv:  NewMockConverter[*wsdom.Workspace, *v1alpha1.Project](ctrl),
	}
}

func (m *bsMocks) handler() *BlockStorageHandler {
	return NewBlockStorageHandler(m.wsRepo, m.skuRepo, m.bsRepo, m.prjRepo, m.bsConv, m.wsConv)
}

// activeProject returns a converted Aruba Project already in the active phase,
// so the Aruba dependency resolution for a block storage succeeds.
func activeProject() *v1alpha1.Project {
	return &v1alpha1.Project{
		Status: v1alpha1.ResourceStatus{Phase: v1alpha1.ResourcePhaseActive},
	}
}

// blockStorage returns a converted Aruba BlockStorage with the given size and
// phase.
func blockStorage(size int32, phase v1alpha1.ResourcePhase) *v1alpha1.BlockStorage {
	return &v1alpha1.BlockStorage{
		Spec: v1alpha1.BlockStorageSpec{SizeGB: size},
		Status: v1alpha1.BlockStorageStatus{
			ResourceStatus: v1alpha1.ResourceStatus{Phase: phase},
		},
	}
}

// expectWorkspaceActive makes the workspace reader report an active workspace,
// which the block storage create flow requires before it proceeds.
func expectWorkspaceActive(m *MockReaderRepo[*wsdom.Workspace]) {
	m.EXPECT().
		Load(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ws **wsdom.Workspace) error {
			(*ws).Status = &wsdom.WorkspaceStatus{
				Status: refdom.Status{State: refdom.ResourceStateActive},
			}
			return nil
		}).
		AnyTimes()
}

func TestBlockStorage_create(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*bsMocks)
		wantErr     bool
		errContains string
	}{
		{
			// The create flow is non-blocking: while a dependency (the
			// workspace) is not ready, it reports "still processing" so the
			// reconciler requeues instead of waiting.
			name: "workspace not ready - still processing",
			setupMocks: func(m *bsMocks) {
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
			setupMocks: func(m *bsMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.skuRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.wsConv.EXPECT().
					FromSECAToAruba(gomock.Any()).
					Return(nil, fmt.Errorf("conversion error"))
			},
			wantErr:     true,
			errContains: "conversion error",
		},
		{
			name: "project not ready - still processing",
			setupMocks: func(m *bsMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.skuRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.bsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(blockStorage(100, v1alpha1.ResourcePhaseActive), nil).AnyTimes()
				// The backing Aruba Project does not exist yet.
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("project")).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "create error",
			setupMocks: func(m *bsMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.skuRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.bsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(blockStorage(100, v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				// Not created yet, so the check reports "not present".
				m.bsRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("block-storage")).AnyTimes()
				m.bsRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(fmt.Errorf("creation error"))
			},
			wantErr:     true,
			errContains: "creation error",
		},
		{
			name: "pending creation - still processing",
			setupMocks: func(m *bsMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.skuRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.bsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(blockStorage(100, v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				// Present but not active yet, so the check reports "not done".
				m.bsRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.bsRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "create idempotent on already exists - still processing",
			setupMocks: func(m *bsMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.skuRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.bsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(blockStorage(100, v1alpha1.ResourcePhaseCreating), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.bsRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				// A re-issued create on an already existing resource must be
				// tolerated (idempotent propagate), so the flow still reports
				// "still processing" rather than a hard error.
				m.bsRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(alreadyExistsErr("block-storage")).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "success create",
			setupMocks: func(m *bsMocks) {
				expectWorkspaceActive(m.wsRepo)
				m.skuRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(activeProject(), nil).AnyTimes()
				m.bsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(blockStorage(100, v1alpha1.ResourcePhaseActive), nil).AnyTimes()
				m.prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				// Present and active, so the check reports "done".
				m.bsRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.bsRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := newBsMocks(ctrl)

			bd := &bsdomblock.BlockStorage{
				Spec: bsdomblock.BlockStorageSpec{
					SizeGB: 100,
					SkuRef: refdom.Reference{Resource: "storage/sku-id"},
				},
			}

			tt.setupMocks(m)

			err := m.handler().Create(context.Background(), bd)

			assertErr(t, err, tt.wantErr, tt.errContains)
		})
	}
}

func TestBlockStorage_delete(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*bsMocks)
		wantErr     bool
		errContains string
	}{
		{
			name: "conversion error",
			setupMocks: func(m *bsMocks) {
				m.bsConv.EXPECT().
					FromSECAToAruba(gomock.Any()).
					Return(nil, fmt.Errorf("conversion error"))
			},
			wantErr:     true,
			errContains: "conversion error",
		},
		{
			name: "delete error",
			setupMocks: func(m *bsMocks) {
				m.bsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(blockStorage(100, v1alpha1.ResourcePhaseDeleting), nil).AnyTimes()
				// Still present, so the check reports "not done".
				m.bsRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.bsRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(fmt.Errorf("deletion error"))
			},
			wantErr:     true,
			errContains: "deletion error",
		},
		{
			name: "pending deletion - still processing",
			setupMocks: func(m *bsMocks) {
				m.bsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(blockStorage(100, v1alpha1.ResourcePhaseDeleting), nil).AnyTimes()
				m.bsRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.bsRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "success delete",
			setupMocks: func(m *bsMocks) {
				m.bsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(blockStorage(100, v1alpha1.ResourcePhaseDeleted), nil).AnyTimes()
				// Gone, so the check reports "done".
				m.bsRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr("block-storage")).AnyTimes()
				m.bsRepo.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := newBsMocks(ctrl)

			bd := &bsdomblock.BlockStorage{
				Spec: bsdomblock.BlockStorageSpec{
					SizeGB: 100,
					SkuRef: refdom.Reference{Resource: "storage/sku-id"},
				},
			}

			tt.setupMocks(m)

			err := m.handler().Delete(context.Background(), bd)

			assertErr(t, err, tt.wantErr, tt.errContains)
		})
	}
}

func TestBlockStorage_increaseSize(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*bsMocks)
		wantErr     bool
		errContains string
	}{
		{
			name: "success resize",
			setupMocks: func(m *bsMocks) {
				// Already resized and active, so the check reports "done".
				m.bsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(blockStorage(200, v1alpha1.ResourcePhaseActive), nil).AnyTimes()
				m.bsRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.bsRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr: false,
		},
		{
			name: "pending resize - still processing",
			setupMocks: func(m *bsMocks) {
				// Right size but still updating, so the check reports "not done".
				m.bsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(blockStorage(200, v1alpha1.ResourcePhaseUpdating), nil).AnyTimes()
				m.bsRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.bsRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			wantErr:     true,
			errContains: "operation still in progress",
		},
		{
			name: "update error",
			setupMocks: func(m *bsMocks) {
				m.bsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(blockStorage(200, v1alpha1.ResourcePhaseUpdating), nil).AnyTimes()
				m.bsRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
				m.bsRepo.EXPECT().Update(gomock.Any(), gomock.Any()).Return(fmt.Errorf("update error"))
			},
			wantErr:     true,
			errContains: "update error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			m := newBsMocks(ctrl)

			bd := &bsdomblock.BlockStorage{
				Spec: bsdomblock.BlockStorageSpec{
					SizeGB: 200, // New size for the increase.
					SkuRef: refdom.Reference{Resource: "storage/sku-id"},
				},
			}

			tt.setupMocks(m)

			err := m.handler().IncreaseSize(context.Background(), bd)

			assertErr(t, err, tt.wantErr, tt.errContains)
		})
	}
}

func assertErr(t *testing.T, err error, wantErr bool, errContains string) {
	t.Helper()

	if wantErr {
		require.Error(t, err)
		if errContains != "" {
			require.Contains(t, err.Error(), errContains)
		}
		return
	}

	require.NoError(t, err)
}
