package crossplane

import (
	"context"
	"testing"

	ionosv1alpha1 "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	skuk8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku/backend/kubernetes"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
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

// TestReadSKU proves readSKU derives the CR namespace from the reference itself
// (tenant+workspace), not from a tenant-only namespace passed in by the caller: the
// SECA gateway writes ECP CRs (InstanceSKU included) at hash(tenant/workspace).
func TestReadSKU(t *testing.T) {
	const (
		tenant    = "tenant-1"
		workspace = "workspace-1"
	)
	skuRef := commondomain.Reference{Resource: "instance-sku/DXS", Workspace: workspace}
	ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: tenant, Workspace: workspace})

	sku := &skuk8s.InstanceSKU{
		ObjectMeta: metav1.ObjectMeta{Name: "DXS", Namespace: ns},
		Spec:       skuk8s.InstanceSkuSpec{VCPU: 2, Ram: 4},
	}
	c := fakeclient.NewClientBuilder().WithScheme(skuScheme(t)).WithObjects(sku).Build()
	b := &base{client: c, logger: testLogger()}

	cores, ramMB, err := b.readSKU(context.Background(), skuRef, tenant)
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
// converter instead of reading raw, un-hashed CR labels. Both CRs live in the
// tenant/workspace namespace derived from their own references, matching how the
// gateway actually writes them.
func TestReadBootImageAlias(t *testing.T) {
	const (
		tenant    = "tenant-1"
		workspace = "workspace-1"
	)

	// Image is a tenant-scoped catalog resource: ImageToCR always places it at
	// hash(tenant) (tenantOnlyScope), regardless of any workspace on the referencing
	// side. Real SourceImageRefs therefore carry no Workspace, matching how the
	// framework's own commonbackend.ReferenceResolver.State resolves SourceImageRef
	// (see resource/storage/v1/block-storage/backend/kubernetes/plugin_handler.go).
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
	bs.Scope = resource.Scope{Tenant: tenant, Workspace: workspace}
	bs.Spec.SizeGB = 42
	bs.Spec.SourceImageRef = &commondomain.Reference{Resource: "image/image-1"}

	bsCR, err := bsk8s.BlockStorageToCR(bs)
	if err != nil {
		t.Fatalf("BlockStorageToCR: %v", err)
	}

	ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: tenant, Workspace: workspace})
	imgNs := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: tenant})
	if imgCR.GetNamespace() != imgNs {
		t.Fatalf("expected image CR in tenant-only namespace %q, got %q", imgNs, imgCR.GetNamespace())
	}
	if bsCR.GetNamespace() != ns {
		t.Fatalf("expected block storage CR in namespace %q, got %q", ns, bsCR.GetNamespace())
	}

	c := fakeclient.NewClientBuilder().WithScheme(imageBlockStorageScheme(t)).WithObjects(imgCR, bsCR).Build()
	b := &base{client: c, logger: testLogger()}

	bootRef := commondomain.Reference{Resource: "block-storage/block-storage-1", Workspace: workspace}
	alias, sizeGB, err := b.readBootImageAlias(context.Background(), bootRef, tenant)
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

func nicNetworkingScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := nick8s.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := ionosv1alpha1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

// TestReadNicNetworking seeds the NIC CR the way production writes it (i.e. through
// NicToCR) to prove readNicNetworking resolves the LAN name from the subnet
// reference's path (".../networks/<network>/subnets/<name>") without reading a Subnet
// CR at all. It covers both the reserved-public-IP path and the DHCP fallback path
// (nic.Spec.PublicIpRefs empty), which the terraform-backed IONOS provider hits in
// practice since it cannot associate a public IP with a NIC.
func TestReadNicNetworking(t *testing.T) {
	const (
		tenant    = "tenant-1"
		workspace = "workspace-1"
		lanName   = "lan-1"
	)
	nicRef := commondomain.Reference{Resource: "nic/nic-1", Workspace: workspace}
	ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: tenant, Workspace: workspace})
	// The IPBlock is a crossplane CR created by the PublicIP plugin at hash(tenant).
	ipNs := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: tenant})

	newNicCR := func(t *testing.T, publicIPRefs []commondomain.Reference) client.Object {
		t.Helper()
		n := &nicdom.Nic{}
		n.Name = "nic-1"
		n.Scope = resource.Scope{Tenant: tenant, Workspace: workspace}
		n.Spec.SubnetRef = commondomain.Reference{Resource: "networks/" + lanName + "/subnets/subnet-1"}
		n.Spec.PublicIpRefs = publicIPRefs

		cr, err := nick8s.NicToCR(n)
		if err != nil {
			t.Fatalf("NicToCR: %v", err)
		}
		cr.SetNamespace(ns)
		return cr
	}

	t.Run("with a reserved public IP", func(t *testing.T) {
		const wantIP = "203.0.113.10"

		ipb := &ionosv1alpha1.Ipblock{
			ObjectMeta: metav1.ObjectMeta{Name: "public-ip-1", Namespace: ipNs},
			Status: ionosv1alpha1.IpblockStatus{
				AtProvider: ionosv1alpha1.IpblockObservation{Ips: []*string{new(wantIP)}},
			},
		}
		nicCR := newNicCR(t, []commondomain.Reference{{Resource: "ipblock/public-ip-1"}})

		c := fakeclient.NewClientBuilder().
			WithScheme(nicNetworkingScheme(t)).
			WithStatusSubresource(&ionosv1alpha1.Ipblock{}).
			WithObjects(nicCR, ipb).
			Build()
		b := &base{client: c, logger: testLogger()}

		gotLan, gotIP, err := b.readNicNetworking(context.Background(), nicRef, tenant)
		if err != nil {
			t.Fatal(err)
		}
		if gotLan != lanName {
			t.Fatalf("readNicNetworking lan = %q, want %q", gotLan, lanName)
		}
		if gotIP != wantIP {
			t.Fatalf("readNicNetworking publicIP = %q, want %q", gotIP, wantIP)
		}
	})

	t.Run("DHCP fallback (no public IP)", func(t *testing.T) {
		nicCR := newNicCR(t, nil)

		c := fakeclient.NewClientBuilder().
			WithScheme(nicNetworkingScheme(t)).
			WithObjects(nicCR).
			Build()
		b := &base{client: c, logger: testLogger()}

		gotLan, gotIP, err := b.readNicNetworking(context.Background(), nicRef, tenant)
		if err != nil {
			t.Fatal(err)
		}
		if gotLan != lanName {
			t.Fatalf("readNicNetworking lan = %q, want %q", gotLan, lanName)
		}
		if gotIP != "" {
			t.Fatalf("readNicNetworking publicIP = %q, want empty (DHCP fallback)", gotIP)
		}
	})
}

// TestNetworkFromSubnetRef directly exercises the subnet-reference parser used by
// readNicNetworking to derive the LAN name without reading a Subnet CR.
func TestNetworkFromSubnetRef(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		ref := commondomain.Reference{Resource: "networks/net-1/subnets/subnet-1"}
		got, err := networkFromSubnetRef(ref)
		if err != nil {
			t.Fatal(err)
		}
		if got != "net-1" {
			t.Fatalf("networkFromSubnetRef = %q, want %q", got, "net-1")
		}
	})

	t.Run("missing network segment", func(t *testing.T) {
		ref := commondomain.Reference{Resource: "subnets/subnet-1"}
		_, err := networkFromSubnetRef(ref)
		if err == nil {
			t.Fatal("networkFromSubnetRef: expected error, got nil")
		}
	})
}
