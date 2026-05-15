package crossplane

import (
	"testing"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

func TestNewDatacenter_name_and_namespace(t *testing.T) {
	d := &regional.WorkspaceDomain{}
	d.Name = "my-workspace"
	d.Scope = scope.Scope{Tenant: "my-tenant"}

	dc := newDatacenter(d)

	if dc.Name != "my-workspace" {
		t.Errorf("expected name %q, got %q", "my-workspace", dc.Name)
	}
	if dc.Spec.ForProvider.Name == nil || *dc.Spec.ForProvider.Name != "my-workspace" {
		t.Errorf("expected ForProvider.Name %q", "my-workspace")
	}
}

func TestNewVolume_name_size_and_datacenter_ref(t *testing.T) {
	d := &regional.BlockStorageDomain{}
	d.Name = "my-volume"
	d.Scope = scope.Scope{Tenant: "t1", Workspace: "ws1"}
	d.Spec.SizeGB = 50

	vol := newVolume(d)

	if vol.Name != "my-volume" {
		t.Errorf("expected name %q, got %q", "my-volume", vol.Name)
	}
	if vol.Spec.ForProvider.Size == nil || *vol.Spec.ForProvider.Size != 50.0 {
		t.Errorf("expected size 50, got %v", vol.Spec.ForProvider.Size)
	}
	if vol.Spec.ForProvider.DatacenterIDRef == nil || vol.Spec.ForProvider.DatacenterIDRef.Name != "ws1" {
		t.Errorf("expected datacenter ref name %q", "ws1")
	}
}
