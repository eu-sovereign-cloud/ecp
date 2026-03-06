package v1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage"
)

func TestBlockStorage_AddToScheme(t *testing.T) {
	s := runtime.NewScheme()
	if err := storage.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme returned unexpected error: %v", err)
	}
}

func TestBlockStorage_GroupRegistered(t *testing.T) {
	s := runtime.NewScheme()
	if err := storage.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme returned unexpected error: %v", err)
	}
	if !s.IsGroupRegistered(storage.Group) {
		t.Errorf("group %q not registered after AddToScheme", storage.Group)
	}
}
