package converter_test

import (
	"testing"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/converter"
)

func TestBlockStorageConverter_FromSECAToAruba(t *testing.T) {
	converter := &converter.BlockStorageConverter{}

	domain := &regional.BlockStorageDomain{
		Spec: regional.BlockStorageSpec{
			SizeGB: 100,
			SkuRef: regional.ReferenceObject{
				Provider: "aruba",
				Region:   "eu-de",
				Resource: "block-storage/silver",
			},
			SourceImageRef: &regional.ReferenceObject{
				Provider:  "aruba",
				Region:    "eu-de",
				Resource:  "image/ubuntu-20.04",
				Workspace: "workspace-abc",
				Tenant:    "tenant-123",
			},
		},
		Status: &regional.BlockStorageStatus{},
	}

	arubaResource, err := converter.FromSECAToAruba(domain)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if int(arubaResource.Spec.SizeGb) != domain.Spec.SizeGB {
		t.Errorf("Expected SizeGB %d, got %d", domain.Spec.SizeGB, arubaResource.Spec.SizeGb)
	}

	// Additional assertions can be added here to verify other fields

}
