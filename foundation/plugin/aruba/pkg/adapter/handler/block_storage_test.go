package handler

import (
"context"
"fmt"
"testing"

"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
"go.uber.org/mock/gomock"
apierrors "k8s.io/apimachinery/pkg/api/errors"
metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
"k8s.io/apimachinery/pkg/runtime/schema"

"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
gwport "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

// ----------------------------------------------------------------------------
// minimal mockReaderRepo — implements gwport.ReaderRepo[T] without mockgen
// ----------------------------------------------------------------------------

type mockReaderRepo[T gwport.IdentifiableResource] struct {
loadFn func(ctx context.Context, m *T) error
}

func (r *mockReaderRepo[T]) Load(ctx context.Context, m *T) error {
if r.loadFn != nil {
return r.loadFn(ctx, m)
}
return nil
}

func (r *mockReaderRepo[T]) List(_ context.Context, _ model.ListParams, _ *[]T) (*string, error) {
return nil, nil
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

func newTestBlockStorageDomain() *regional.BlockStorageDomain {
return &regional.BlockStorageDomain{
Metadata: regional.Metadata{
CommonMetadata: model.CommonMetadata{Name: "my-bs"},
Scope: scope.Scope{
Tenant:    "test-tenant",
Workspace: "test-ws",
},
},
Spec: regional.BlockStorageSpec{
SizeGB: 100,
SkuRef: regional.ReferenceObject{
Resource: "storageClass/standard",
Tenant:   "test-tenant",
},
},
}
}

func newActiveWorkspace() *regional.WorkspaceDomain {
state := regional.ResourceStateActive
return &regional.WorkspaceDomain{
Metadata: regional.Metadata{
CommonMetadata: model.CommonMetadata{Name: "test-ws"},
Scope:          scope.Scope{Tenant: "test-tenant"},
},
Status: &regional.WorkspaceStatusDomain{
StatusDomain: regional.StatusDomain{State: &state},
},
}
}

// buildHandler constructs a BlockStorageHandler with all 6 dependencies injected.
func buildHandler(
ctrl *gomock.Controller,
wsLoad func(context.Context, **regional.WorkspaceDomain) error,
skuLoad func(context.Context, **regional.StorageSKUDomain) error,
bsRepo *MockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList],
prjRepo *MockRepository[*v1alpha1.Project, *v1alpha1.ProjectList],
bsConv *MockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage],
wsConv *MockConverter[*regional.WorkspaceDomain, *v1alpha1.Project],
) *BlockStorageHandler {
wsRepo := &mockReaderRepo[*regional.WorkspaceDomain]{loadFn: wsLoad}
skuRepo := &mockReaderRepo[*regional.StorageSKUDomain]{loadFn: skuLoad}
return NewBlockStorageHandler(wsRepo, skuRepo, bsRepo, prjRepo, bsConv, wsConv)
}

// ----------------------------------------------------------------------------
// blockStorageMutateSizeFunc — pure logic, no mocks needed
// ----------------------------------------------------------------------------

func TestBlockStorageHandler_MutateSizeFunc(t *testing.T) {
ctrl := gomock.NewController(t)
defer ctrl.Finish()

bsRepo := NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl)
prjRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
bsConv := NewMockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage](ctrl)
wsConv := NewMockConverter[*regional.WorkspaceDomain, *v1alpha1.Project](ctrl)

h := buildHandler(ctrl, nil, nil, bsRepo, prjRepo, bsConv, wsConv)

mutable := &ArubaBlockStorageBundle{
BlockStorage: &v1alpha1.BlockStorage{Spec: v1alpha1.BlockStorageSpec{SizeGb: 10}},
}
params := &SecaBlockStorageBundle{
BlockStorage: &regional.BlockStorageDomain{Spec: regional.BlockStorageSpec{SizeGB: 200}},
}

err := h.blockStorageMutateSizeFunc(mutable, params)
require.NoError(t, err)
assert.Equal(t, int32(200), mutable.BlockStorage.Spec.SizeGb)
}

// ----------------------------------------------------------------------------
// checkBsDeleteCondition
// ----------------------------------------------------------------------------

func TestBlockStorageHandler_CheckBsDeleteCondition_NotFound(t *testing.T) {
ctrl := gomock.NewController(t)
defer ctrl.Finish()

notFoundErr := apierrors.NewNotFound(schema.GroupResource{Group: "arubacloud.com", Resource: "blockstorage"}, "my-bs")

bsRepo := NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl)
bsRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(notFoundErr).AnyTimes()

prjRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
bsConv := NewMockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage](ctrl)
wsConv := NewMockConverter[*regional.WorkspaceDomain, *v1alpha1.Project](ctrl)

h := buildHandler(ctrl, nil, nil, bsRepo, prjRepo, bsConv, wsConv)

bundle := &ArubaBlockStorageBundle{BlockStorage: &v1alpha1.BlockStorage{ObjectMeta: metav1.ObjectMeta{Name: "my-bs"}}}
assert.True(t, h.checkBsDeleteCondition(bundle), "should return true when resource is not found")
}

func TestBlockStorageHandler_CheckBsDeleteCondition_StillExists(t *testing.T) {
ctrl := gomock.NewController(t)
defer ctrl.Finish()

bsRepo := NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl)
bsRepo.EXPECT().Load(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

prjRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
bsConv := NewMockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage](ctrl)
wsConv := NewMockConverter[*regional.WorkspaceDomain, *v1alpha1.Project](ctrl)

h := buildHandler(ctrl, nil, nil, bsRepo, prjRepo, bsConv, wsConv)

bundle := &ArubaBlockStorageBundle{BlockStorage: &v1alpha1.BlockStorage{ObjectMeta: metav1.ObjectMeta{Name: "my-bs"}}}
assert.False(t, h.checkBsDeleteCondition(bundle), "should return false when resource still exists")
}

// ----------------------------------------------------------------------------
// BypassDependencyResolver
// ----------------------------------------------------------------------------

func TestBlockStorageHandler_BypassDependencyResolver(t *testing.T) {
ctrl := gomock.NewController(t)
defer ctrl.Finish()

bsRepo := NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl)
prjRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
bsConv := NewMockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage](ctrl)
wsConv := NewMockConverter[*regional.WorkspaceDomain, *v1alpha1.Project](ctrl)

h := buildHandler(ctrl, nil, nil, bsRepo, prjRepo, bsConv, wsConv)

resource := newTestBlockStorageDomain()
bundle, err := h.BypassDependencyResolver(context.Background(), resource)

require.NoError(t, err)
assert.Equal(t, resource, bundle.BlockStorage)
assert.Nil(t, bundle.Workspace)
assert.Nil(t, bundle.StorageSku)
}

// ----------------------------------------------------------------------------
// resolveSecaBlockStorageDependencies
// ----------------------------------------------------------------------------

func TestBlockStorageHandler_ResolveSecaDeps_WorkspaceNotFound(t *testing.T) {
ctrl := gomock.NewController(t)
defer ctrl.Finish()

bsRepo := NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl)
prjRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
bsConv := NewMockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage](ctrl)
wsConv := NewMockConverter[*regional.WorkspaceDomain, *v1alpha1.Project](ctrl)

loadWSErr := fmt.Errorf("workspace not found")
h := buildHandler(ctrl,
func(_ context.Context, _ **regional.WorkspaceDomain) error { return loadWSErr },
nil, bsRepo, prjRepo, bsConv, wsConv)

resource := newTestBlockStorageDomain()
_, err := h.resolveSecaBlockStorageDependencies(context.Background(), resource)

require.Error(t, err)
}

func TestBlockStorageHandler_ResolveSecaDeps_WorkspaceNotActive(t *testing.T) {
ctrl := gomock.NewController(t)
defer ctrl.Finish()

bsRepo := NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl)
prjRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
bsConv := NewMockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage](ctrl)
wsConv := NewMockConverter[*regional.WorkspaceDomain, *v1alpha1.Project](ctrl)

pendingState := regional.ResourceStatePending
h := buildHandler(ctrl,
func(_ context.Context, m **regional.WorkspaceDomain) error {
(*m).Status = &regional.WorkspaceStatusDomain{
StatusDomain: regional.StatusDomain{State: &pendingState},
}
return nil
},
nil, bsRepo, prjRepo, bsConv, wsConv)

resource := newTestBlockStorageDomain()
_, err := h.resolveSecaBlockStorageDependencies(context.Background(), resource)

require.Error(t, err, "should fail when workspace is not active")
}

func TestBlockStorageHandler_ResolveSecaDeps_InvalidSKURef(t *testing.T) {
ctrl := gomock.NewController(t)
defer ctrl.Finish()

bsRepo := NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl)
prjRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
bsConv := NewMockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage](ctrl)
wsConv := NewMockConverter[*regional.WorkspaceDomain, *v1alpha1.Project](ctrl)

activeWS := newActiveWorkspace()
h := buildHandler(ctrl,
func(_ context.Context, m **regional.WorkspaceDomain) error { *m = activeWS; return nil },
nil, bsRepo, prjRepo, bsConv, wsConv)

resource := newTestBlockStorageDomain()
resource.Spec.SkuRef.Resource = "no-slash-here" // invalid — must be "class/name"

_, err := h.resolveSecaBlockStorageDependencies(context.Background(), resource)
require.Error(t, err)
assert.Contains(t, err.Error(), "invalid SKU reference")
}

func TestBlockStorageHandler_ResolveSecaDeps_Success(t *testing.T) {
ctrl := gomock.NewController(t)
defer ctrl.Finish()

bsRepo := NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl)
prjRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
bsConv := NewMockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage](ctrl)
wsConv := NewMockConverter[*regional.WorkspaceDomain, *v1alpha1.Project](ctrl)

activeWS := newActiveWorkspace()
sku := &regional.StorageSKUDomain{}

h := buildHandler(ctrl,
func(_ context.Context, m **regional.WorkspaceDomain) error { *m = activeWS; return nil },
func(_ context.Context, m **regional.StorageSKUDomain) error { *m = sku; return nil },
bsRepo, prjRepo, bsConv, wsConv)

resource := newTestBlockStorageDomain()
bundle, err := h.resolveSecaBlockStorageDependencies(context.Background(), resource)

require.NoError(t, err)
assert.Equal(t, resource, bundle.BlockStorage)
assert.Equal(t, activeWS, bundle.Workspace)
}

// ----------------------------------------------------------------------------
// Create — happy path end-to-end
// ----------------------------------------------------------------------------

func TestBlockStorageHandler_Create_HappyPath(t *testing.T) {
ctrl := gomock.NewController(t)
defer ctrl.Finish()

bsRepo := NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl)
prjRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
bsConv := NewMockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage](ctrl)
wsConv := NewMockConverter[*regional.WorkspaceDomain, *v1alpha1.Project](ctrl)

activeWS := newActiveWorkspace()
sku := &regional.StorageSKUDomain{}

arubaBS := &v1alpha1.BlockStorage{
ObjectMeta: metav1.ObjectMeta{Name: "my-bs", Namespace: "test-ns"},
Status: v1alpha1.BlockStorageStatus{
ResourceStatus: v1alpha1.ResourceStatus{Phase: v1alpha1.ResourcePhaseCreated},
},
}
arubaProject := &v1alpha1.Project{
Status: v1alpha1.ResourceStatus{Phase: v1alpha1.ResourcePhaseCreated},
}

bsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(arubaBS, nil).AnyTimes()
wsConv.EXPECT().FromSECAToAruba(gomock.Any()).Return(arubaProject, nil).AnyTimes()

prjRepo.EXPECT().Load(gomock.Any(), gomock.Any()).
DoAndReturn(func(_ context.Context, p *v1alpha1.Project) error {
p.Status = v1alpha1.ResourceStatus{Phase: v1alpha1.ResourcePhaseCreated}
return nil
}).AnyTimes()
bsRepo.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
bsRepo.EXPECT().WaitUntil(gomock.Any(), gomock.Any(), gomock.Any()).
DoAndReturn(func(_ context.Context, _ *v1alpha1.BlockStorage, cond func(*v1alpha1.BlockStorage) bool) (*v1alpha1.BlockStorage, error) {
if cond(arubaBS) {
return arubaBS, nil
}
return nil, fmt.Errorf("condition not met")
})

h := buildHandler(ctrl,
func(_ context.Context, m **regional.WorkspaceDomain) error { *m = activeWS; return nil },
func(_ context.Context, m **regional.StorageSKUDomain) error { *m = sku; return nil },
bsRepo, prjRepo, bsConv, wsConv)

err := h.Create(context.Background(), newTestBlockStorageDomain())
require.NoError(t, err)
}

// ----------------------------------------------------------------------------
// Create — workspace dependency error propagates
// ----------------------------------------------------------------------------

func TestBlockStorageHandler_Create_WorkspaceNotReady(t *testing.T) {
ctrl := gomock.NewController(t)
defer ctrl.Finish()

bsRepo := NewMockRepository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList](ctrl)
prjRepo := NewMockRepository[*v1alpha1.Project, *v1alpha1.ProjectList](ctrl)
bsConv := NewMockConverter[*regional.BlockStorageDomain, *v1alpha1.BlockStorage](ctrl)
wsConv := NewMockConverter[*regional.WorkspaceDomain, *v1alpha1.Project](ctrl)

h := buildHandler(ctrl,
func(_ context.Context, _ **regional.WorkspaceDomain) error {
return fmt.Errorf("workspace not found")
},
nil, bsRepo, prjRepo, bsConv, wsConv)

err := h.Create(context.Background(), newTestBlockStorageDomain())
require.Error(t, err)
}
