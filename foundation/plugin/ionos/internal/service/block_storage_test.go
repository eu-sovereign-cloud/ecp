package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	blockstoragectrl "github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/internal/controller/block_storage"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/internal/service"
)

type stubBlockStorageStore struct {
	createErr       error
	deleteErr       error
	increaseSizeErr error
}

func (s *stubBlockStorageStore) Create(_ context.Context, _ *regional.BlockStorageDomain) error {
	return s.createErr
}

func (s *stubBlockStorageStore) Delete(_ context.Context, _ *regional.BlockStorageDomain) error {
	return s.deleteErr
}

func (s *stubBlockStorageStore) IncreaseSize(_ context.Context, _ *regional.BlockStorageDomain) error {
	return s.increaseSizeErr
}

func bsDomain(name, tenant, ws string, sizeGB int) *regional.BlockStorageDomain {
	d := &regional.BlockStorageDomain{}
	d.Name = name
	d.Scope = scope.Scope{Tenant: tenant, Workspace: ws}
	d.Spec.SizeGB = sizeGB
	return d
}

func newBlockStorageService(store *stubBlockStorageStore) *service.BlockStorage {
	return &service.BlockStorage{
		Creator:       &blockstoragectrl.CreateBlockStorage{Store: store},
		Deleter:       &blockstoragectrl.DeleteBlockStorage{Store: store},
		SizeIncreaser: &blockstoragectrl.IncreaseSizeBlockStorage{Store: store},
	}
}

func TestBlockStorageService_Create_delegates(t *testing.T) {
	if err := newBlockStorageService(&stubBlockStorageStore{}).Create(context.Background(), bsDomain("v", "t1", "ws1", 10)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBlockStorageService_Create_propagates_error(t *testing.T) {
	want := errors.New("create error")
	got := newBlockStorageService(&stubBlockStorageStore{createErr: want}).Create(context.Background(), bsDomain("v", "t1", "ws1", 10))
	if !errors.Is(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestBlockStorageService_Delete_delegates(t *testing.T) {
	if err := newBlockStorageService(&stubBlockStorageStore{}).Delete(context.Background(), bsDomain("v", "t1", "ws1", 10)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBlockStorageService_IncreaseSize_delegates(t *testing.T) {
	if err := newBlockStorageService(&stubBlockStorageStore{}).IncreaseSize(context.Background(), bsDomain("v", "t1", "ws1", 20)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBlockStorageService_IncreaseSize_propagates_error(t *testing.T) {
	want := errors.New("increase size error")
	got := newBlockStorageService(&stubBlockStorageStore{increaseSizeErr: want}).IncreaseSize(context.Background(), bsDomain("v", "t1", "ws1", 20))
	if !errors.Is(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
