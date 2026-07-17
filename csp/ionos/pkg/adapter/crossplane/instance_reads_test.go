package crossplane

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	skuk8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku/backend/kubernetes"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
)

func skuScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := skuk8s.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestReadSKU(t *testing.T) {
	sku := &skuk8s.InstanceSKU{
		ObjectMeta: metav1.ObjectMeta{Name: "DXS", Namespace: "tenant-1"},
		Spec:       skuk8s.InstanceSkuSpec{VCPU: 2, Ram: 4},
	}
	c := fakeclient.NewClientBuilder().WithScheme(skuScheme(t)).WithObjects(sku).Build()
	b := &base{client: c, logger: testLogger()}

	cores, ramMB, err := b.readSKU(context.Background(), "tenant-1", "DXS")
	if err != nil {
		t.Fatal(err)
	}
	if cores != 2 || ramMB != 4096 {
		t.Fatalf("readSKU = (%v cores, %v MB), want (2, 4096)", cores, ramMB)
	}
}

func imageBlockStorageScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := imgk8s.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := bsk8s.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

// TestReadBootImageAlias seeds the Image and BlockStorage CRs the way production writes
// them (i.e. through ImageToCR/BlockStorageToCR, which hash user label keys via
// OriginalToKeyed) to prove readBootImageAlias resolves labels through the domain
// converter instead of reading raw, un-hashed CR labels.
func TestReadBootImageAlias(t *testing.T) {
	tenant := "tenant-1"

	img := &imgdom.Image{}
	img.Name = "image-1"
	img.Scope = resource.Scope{Tenant: tenant}
	img.Labels = map[string]string{"base": "ubuntu", "version": "24.04"}

	imgCR, err := imgk8s.ImageToCR(img)
	if err != nil {
		t.Fatalf("ImageToCR: %v", err)
	}

	bs := &bsdom.BlockStorage{}
	bs.Name = "block-storage-1"
	bs.Scope = resource.Scope{Tenant: tenant}
	bs.Spec.SizeGB = 42
	bs.Spec.SourceImageRef = &commondomain.Reference{Resource: "image/image-1"}

	bsCR, err := bsk8s.BlockStorageToCR(bs)
	if err != nil {
		t.Fatalf("BlockStorageToCR: %v", err)
	}

	ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: tenant})
	if imgCR.GetNamespace() != ns || bsCR.GetNamespace() != ns {
		t.Fatalf("expected both CRs in namespace %q, got image=%q blockstorage=%q", ns, imgCR.GetNamespace(), bsCR.GetNamespace())
	}

	c := fakeclient.NewClientBuilder().WithScheme(imageBlockStorageScheme(t)).WithObjects(imgCR, bsCR).Build()
	b := &base{client: c, logger: testLogger()}

	alias, sizeGB, err := b.readBootImageAlias(context.Background(), ns, "block-storage-1")
	if err != nil {
		t.Fatal(err)
	}
	if alias != "ubuntu:24.04" {
		t.Fatalf("readBootImageAlias alias = %q, want %q", alias, "ubuntu:24.04")
	}
	if sizeGB != 42 {
		t.Fatalf("readBootImageAlias sizeGB = %d, want 42", sizeGB)
	}
}
