package v1

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/eu-sovereign-cloud/ecp/foundation/api/regional/network"
)

func TestNetworkSKU_AddToScheme(t *testing.T) {
	s := runtime.NewScheme()
	if err := network.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme returned unexpected error: %v", err)
	}
}

func TestNetworkSKU_GroupRegistered(t *testing.T) {
	s := runtime.NewScheme()
	if err := network.AddToScheme(s); err != nil {
		t.Fatalf("AddToScheme returned unexpected error: %v", err)
	}
	if !s.IsGroupRegistered(network.Group) {
		t.Errorf("group %q not registered after AddToScheme", network.Group)
	}
}

func TestNetworkSKU_DeepCopy(t *testing.T) {
	sku := &SKU{}
	copy := sku.DeepCopyObject()
	if copy == nil {
		t.Fatal("DeepCopyObject returned nil")
	}
	if _, ok := copy.(*SKU); !ok {
		t.Fatalf("DeepCopyObject returned unexpected type %T", copy)
	}
}

func TestNetworkSKUList_DeepCopy(t *testing.T) {
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
