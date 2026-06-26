package kubernetes_test

import (
	"strings"
	"testing"

	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"

	. "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
)

// FuzzImageSpecRoundTrip verifies that an Image domain survives a
// domain→CR→domain→CR→domain→CR round-trip. The interesting complexity is in
// the BlockStorageRef reference conversion, which normalises embedded
// "providers/", "regions/", "tenants/", "workspaces/" path segments.
//
// Invariants:
//   - Name, Provider, Tenant, Region and the enum spec fields are stable after one
//     round-trip (domain2 == domain3). Images are tenant-scoped, so Workspace is
//     never set.
//   - BlockStorageRef CR fields are compared at cr2 vs cr3: the first pass normalises
//     embedded path segments; the second pass must produce an identical CR.
func FuzzImageSpecRoundTrip(f *testing.F) {
	// (refResource, refProvider, refRegion, refTenant, refWorkspace, cpuArch, boot, initializer, name, provider, tenant, region)
	f.Add("block-storages/my-bs", "ionos", "de-fra", "t-1", "ws-1", "amd64", "UEFI", "cloudinit-22", "img-1", "ionos/de", "t-1", "de-fra")
	f.Add("", "", "", "", "", "", "", "", "", "", "", "")
	f.Add("block-storages/bs", "", "", "", "", "arm64", "BIOS", "none", "img", "", "t", "")
	// length limits
	f.Add("block-storages/"+strings.Repeat("x", 253), "", "", "", "", "amd64", "UEFI", "none", strings.Repeat("n", 254), "", "t", "")
	// provider/region slash/underscore edge cases
	f.Add("block-storages/bs", "///", "de/fra", "t", "ws", "amd64", "UEFI", "none", "img", "a/_b", "t", "de")
	f.Add("block-storages/bs", "ionos/Muenchen", "eu/central", "t", "ws", "arm64", "BIOS", "cloudinit-22", "img", "provider/nihongo", "t", "de")

	f.Fuzz(func(t *testing.T,
		refResource, refProvider, refRegion, refTenant, refWorkspace string,
		cpuArch, boot, initializer string,
		name, provider, tenant, region string,
	) {
		domain := &imgdom.Image{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{
					Name:     name,
					Provider: provider,
				},
				Scope:  kernelresource.Scope{Tenant: tenant},
				Region: region,
			},
			Spec: imgdom.ImageSpec{
				BlockStorageRef: commondomain.Reference{
					Resource:  refResource,
					Provider:  refProvider,
					Region:    refRegion,
					Tenant:    refTenant,
					Workspace: refWorkspace,
				},
				CpuArchitecture: cpuArch,
				Boot:            boot,
				Initializer:     initializer,
			},
		}

		cr1, err := ImageToCR(domain)
		if err != nil {
			return
		}

		domain2, err := ImageFromCR(cr1)
		if err != nil {
			t.Errorf("CR→domain failed after successful domain→CR: %v", err)
			return
		}

		cr2, err := ImageToCR(domain2)
		if err != nil {
			t.Errorf("second domain→CR failed: %v", err)
			return
		}

		domain3, err := ImageFromCR(cr2)
		if err != nil {
			t.Errorf("second CR→domain failed: %v", err)
			return
		}

		cr3, err := ImageToCR(domain3)
		if err != nil {
			t.Errorf("third domain→CR failed: %v", err)
			return
		}

		// CommonMetadata and enum spec fields: stable after one round-trip.
		if domain2.Name != domain3.Name {
			t.Errorf("Name not stable: %q → %q", domain2.Name, domain3.Name)
		}
		if domain2.Provider != domain3.Provider {
			t.Errorf("Provider not stable: %q → %q", domain2.Provider, domain3.Provider)
		}
		if domain2.Tenant != domain3.Tenant {
			t.Errorf("Tenant not stable: %q → %q", domain2.Tenant, domain3.Tenant)
		}
		if domain2.Region != domain3.Region {
			t.Errorf("Region not stable: %q → %q", domain2.Region, domain3.Region)
		}
		if domain2.Spec.CpuArchitecture != domain3.Spec.CpuArchitecture {
			t.Errorf("CpuArchitecture not stable: %q → %q", domain2.Spec.CpuArchitecture, domain3.Spec.CpuArchitecture)
		}
		if domain2.Spec.Boot != domain3.Spec.Boot {
			t.Errorf("Boot not stable: %q → %q", domain2.Spec.Boot, domain3.Spec.Boot)
		}
		if domain2.Spec.Initializer != domain3.Spec.Initializer {
			t.Errorf("Initializer not stable: %q → %q", domain2.Spec.Initializer, domain3.Spec.Initializer)
		}

		// BlockStorageRef domain-level stability: after the first round-trip normalises
		// embedded path segments, domain2 and domain3 must carry identical values.
		dref2, dref3 := domain2.Spec.BlockStorageRef, domain3.Spec.BlockStorageRef
		if dref2 != dref3 {
			t.Errorf("BlockStorageRef not stable at domain level:\n  pass1: %+v\n  pass2: %+v", dref2, dref3)
		}

		// BlockStorageRef CR-level stability: cr2 and cr3 must be identical (encoded form).
		img2 := cr2.(*Image)
		img3 := cr3.(*Image)
		ref2, ref3 := img2.Spec.BlockStorageRef, img3.Spec.BlockStorageRef
		if ref2.Provider != ref3.Provider {
			t.Errorf("BlockStorageRef.Provider not stable: %q → %q", ref2.Provider, ref3.Provider)
		}
		if ref2.Region != ref3.Region {
			t.Errorf("BlockStorageRef.Region not stable: %q → %q", ref2.Region, ref3.Region)
		}
		if ref2.Tenant != ref3.Tenant {
			t.Errorf("BlockStorageRef.Tenant not stable: %q → %q", ref2.Tenant, ref3.Tenant)
		}
		if ref2.Workspace != ref3.Workspace {
			t.Errorf("BlockStorageRef.Workspace not stable: %q → %q", ref2.Workspace, ref3.Workspace)
		}
		if ref2.Resource != ref3.Resource {
			t.Errorf("BlockStorageRef.Resource not stable: %q → %q", ref2.Resource, ref3.Resource)
		}
	})
}
