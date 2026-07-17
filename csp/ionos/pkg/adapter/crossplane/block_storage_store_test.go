package crossplane

import (
	"context"
	"testing"

	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
)

func bootBlockStorage() *bsdom.BlockStorage {
	b := &bsdom.BlockStorage{}
	b.Name = "block-storage-1"
	b.Scope = resource.Scope{Tenant: "tenant-1", Workspace: "workspace-1"}
	b.Spec.SizeGB = 10
	b.Spec.SourceImageRef = &commondomain.Reference{Resource: "image/image-1"}
	return b
}

// Image-backed BlockStorage must not create a Volume; it observes and waits.
func TestBlockStorageImageBackedIsObserver(t *testing.T) {
	c := fakeclient.NewClientBuilder().WithScheme(ionosScheme(t)).Build()
	store := NewBlockStorageStore(c, testLogger())

	if err := store.Create(context.Background(), bootBlockStorage()); err != nil {
		t.Fatalf("Create = %v, want ErrStillProcessing (waiting for instance)", err)
	}

	// ComputeNamespace(&resource.Scope{Tenant: "tenant-1"}) hashes the tenant with
	// SHA3-224 rather than returning the literal tenant name; compute it directly.
	ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: "tenant-1"})

	vol := &ionosv1alpha1.Volume{}
	err := c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: "block-storage-1"}, vol)
	if err == nil {
		t.Fatal("image-backed BlockStorage must NOT create a Volume, but one exists")
	}
}
