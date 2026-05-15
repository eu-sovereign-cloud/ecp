package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	workspacectrl "github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/internal/controller/workspace"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/internal/service"
)

type stubWorkspaceStore struct {
	createErr error
	deleteErr error
}

func (s *stubWorkspaceStore) Create(_ context.Context, _ *regional.WorkspaceDomain) error {
	return s.createErr
}

func (s *stubWorkspaceStore) Delete(_ context.Context, _ *regional.WorkspaceDomain) error {
	return s.deleteErr
}

func wsDomain(name, tenant string) *regional.WorkspaceDomain {
	d := &regional.WorkspaceDomain{}
	d.Name = name
	d.Scope = scope.Scope{Tenant: tenant}
	return d
}

func newWorkspaceService(store *stubWorkspaceStore) *service.Workspace {
	return &service.Workspace{
		Creator: &workspacectrl.CreateWorkspace{Store: store},
		Deleter: &workspacectrl.DeleteWorkspace{Store: store},
	}
}

func TestWorkspaceService_Create_delegates(t *testing.T) {
	if err := newWorkspaceService(&stubWorkspaceStore{}).Create(context.Background(), wsDomain("ws", "t1")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkspaceService_Create_propagates_error(t *testing.T) {
	want := errors.New("create error")
	got := newWorkspaceService(&stubWorkspaceStore{createErr: want}).Create(context.Background(), wsDomain("ws", "t1"))
	if !errors.Is(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestWorkspaceService_Delete_delegates(t *testing.T) {
	if err := newWorkspaceService(&stubWorkspaceStore{}).Delete(context.Background(), wsDomain("ws", "t1")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkspaceService_Delete_propagates_error(t *testing.T) {
	want := errors.New("delete error")
	got := newWorkspaceService(&stubWorkspaceStore{deleteErr: want}).Delete(context.Background(), wsDomain("ws", "t1"))
	if !errors.Is(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
