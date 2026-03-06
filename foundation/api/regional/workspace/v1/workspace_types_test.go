package v1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/workspace"
)

func TestWorkspace_AddToScheme(t *testing.T) {
	s := runtime.NewScheme()
	if err := workspace.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme returned unexpected error: %v", err)
	}
}

func TestWorkspace_GroupRegistered(t *testing.T) {
	s := runtime.NewScheme()
	if err := workspace.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme returned unexpected error: %v", err)
	}
	if !s.IsGroupRegistered(workspace.Group) {
		t.Errorf("group %q not registered after AddToScheme", workspace.Group)
	}
}

func TestWorkspace_DeepCopy(t *testing.T) {
	ws := &Workspace{}
	copy := ws.DeepCopyObject()
	if copy == nil {
		t.Fatal("DeepCopyObject returned nil")
	}
	if _, ok := copy.(*Workspace); !ok {
		t.Fatalf("DeepCopyObject returned unexpected type %T", copy)
	}
}

func TestWorkspaceList_DeepCopy(t *testing.T) {
	list := &WorkspaceList{
		Items: []Workspace{{}, {}},
	}
	copy := list.DeepCopyObject()
	if copy == nil {
		t.Fatal("DeepCopyObject returned nil")
	}
	copied, ok := copy.(*WorkspaceList)
	if !ok {
		t.Fatalf("DeepCopyObject returned unexpected type %T", copy)
	}
	if len(copied.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(copied.Items))
	}
}
