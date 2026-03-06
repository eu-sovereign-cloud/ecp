package v1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage"
)

func TestStorageSKU_AddToScheme(t *testing.T) {
	s := runtime.NewScheme()
	if err := storage.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme returned unexpected error: %v", err)
	}
}

func TestStorageSKU_GroupRegistered(t *testing.T) {
	s := runtime.NewScheme()
	if err := storage.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme returned unexpected error: %v", err)
	}
	if !s.IsGroupRegistered(storage.Group) {
		t.Errorf("group %q not registered after AddToScheme", storage.Group)
	}
}

func TestStorageSKU_DeepCopy(t *testing.T) {
	sku := &SKU{}
	copy := sku.DeepCopyObject()
	if copy == nil {
		t.Fatal("DeepCopyObject returned nil")
	}
	if _, ok := copy.(*SKU); !ok {
		t.Fatalf("DeepCopyObject returned unexpected type %T", copy)
	}
}

func TestStorageSKUList_DeepCopy(t *testing.T) {
	list := &SKUList{
		Items: []SKU{{}, {}},
	}
	copy := list.DeepCopyObject()
	if copy == nil {
		t.Fatal("DeepCopyObject returned nil")
	}
	copied, ok := copy.(*SKUList)
	if !ok {
		t.Fatalf("DeepCopyObject returned unexpected type %T", copy)
	}
	if len(copied.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(copied.Items))
	}
}
