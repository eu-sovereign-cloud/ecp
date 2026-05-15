package block_storage_test

import (
	"context"
	"errors"
	"testing"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	blockstorage "github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/internal/controller/block_storage"
)

type stubBlockStorageStore struct {
	createErr       error
	deleteErr       error
	increaseSizeErr error
	createCalled    bool
}

func (s *stubBlockStorageStore) Create(_ context.Context, _ *regional.BlockStorageDomain) error {
	s.createCalled = true
	return s.createErr
}

func (s *stubBlockStorageStore) Delete(_ context.Context, _ *regional.BlockStorageDomain) error {
	return s.deleteErr
}

func (s *stubBlockStorageStore) IncreaseSize(_ context.Context, _ *regional.BlockStorageDomain) error {
	return s.increaseSizeErr
}

func bsDomain(name, tenant, workspace string, sizeGB int) *regional.BlockStorageDomain {
	d := &regional.BlockStorageDomain{}
	d.Name = name
	d.Scope = scope.Scope{Tenant: tenant, Workspace: workspace}
	d.Spec.SizeGB = sizeGB
	return d
}

func TestCreateBlockStorage_Do_propagates_error(t *testing.T) {
	want := errors.New("store error")
	store := &stubBlockStorageStore{createErr: want}
	ctrl := &blockstorage.CreateBlockStorage{Store: store}

	got := ctrl.Do(context.Background(), bsDomain("vol", "t1", "ws1", 10))
	if !errors.Is(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestDeleteBlockStorage_Do_calls_store(t *testing.T) {
	store := &stubBlockStorageStore{}
	ctrl := &blockstorage.DeleteBlockStorage{Store: store}

	if err := ctrl.Do(context.Background(), bsDomain("vol", "t1", "ws1", 10)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteBlockStorage_Do_propagates_error(t *testing.T) {
	want := errors.New("delete error")
	store := &stubBlockStorageStore{deleteErr: want}
	ctrl := &blockstorage.DeleteBlockStorage{Store: store}

	got := ctrl.Do(context.Background(), bsDomain("vol", "t1", "ws1", 10))
	if !errors.Is(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestIncreaseSizeBlockStorage_Do_calls_store(t *testing.T) {
	store := &stubBlockStorageStore{}
	ctrl := &blockstorage.IncreaseSizeBlockStorage{Store: store}

	if err := ctrl.Do(context.Background(), bsDomain("vol", "t1", "ws1", 20)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIncreaseSizeBlockStorage_Do_propagates_error(t *testing.T) {
	want := errors.New("increase size error")
	store := &stubBlockStorageStore{increaseSizeErr: want}
	ctrl := &blockstorage.IncreaseSizeBlockStorage{Store: store}

	got := ctrl.Do(context.Background(), bsDomain("vol", "t1", "ws1", 20))
	if !errors.Is(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
