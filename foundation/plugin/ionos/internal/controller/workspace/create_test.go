package workspace_test

import (
	"context"
	"errors"
	"testing"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/ionos/internal/controller/workspace"
)

type stubWorkspaceStore struct {
	createErr    error
	deleteErr    error
	createCalled bool
}

func (s *stubWorkspaceStore) Create(_ context.Context, _ *regional.WorkspaceDomain) error {
	s.createCalled = true
	return s.createErr
}

func (s *stubWorkspaceStore) Delete(_ context.Context, _ *regional.WorkspaceDomain) error {
	return s.deleteErr
}

func domain(name, tenant string) *regional.WorkspaceDomain {
	d := &regional.WorkspaceDomain{}
	d.Name = name
	d.Scope = scope.Scope{Tenant: tenant}
	return d
}

func TestCreateWorkspace_Do_propagates_error(t *testing.T) {
	want := errors.New("store error")
	store := &stubWorkspaceStore{createErr: want}
	ctrl := &workspace.CreateWorkspace{Store: store}

	got := ctrl.Do(context.Background(), domain("ws", "t1"))
	if !errors.Is(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestDeleteWorkspace_Do_calls_store(t *testing.T) {
	store := &stubWorkspaceStore{}
	ctrl := &workspace.DeleteWorkspace{Store: store}

	if err := ctrl.Do(context.Background(), domain("ws", "t1")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteWorkspace_Do_propagates_error(t *testing.T) {
	want := errors.New("delete error")
	store := &stubWorkspaceStore{deleteErr: want}
	ctrl := &workspace.DeleteWorkspace{Store: store}

	got := ctrl.Do(context.Background(), domain("ws", "t1"))
	if !errors.Is(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
