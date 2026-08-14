package crossplane

import (
	"testing"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
)

func TestNewLanIsPublic(t *testing.T) {
	n := &netdom.Network{}
	n.Name = "network-1"
	n.Scope = resource.Scope{Tenant: "tenant-1", Workspace: "workspace-1"}

	lan := newLan(n)
	if lan.Spec.ForProvider.Public == nil || !*lan.Spec.ForProvider.Public {
		t.Fatalf("newLan Public = %v, want true", lan.Spec.ForProvider.Public)
	}
}
