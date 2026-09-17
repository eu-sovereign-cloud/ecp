package converter_test

import (
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
)

// secaBlockStorage is a valid SECA block storage. Its source image reference names a region of
// its own ("eu-de") because a reference carries one only when it points across a region - the
// volume still lands in the region the resource itself carries.
func secaBlockStorage() *bsdom.BlockStorage {
	return &bsdom.BlockStorage{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: "my-block-storage"},
			Scope:          res.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
			Region:         "ITBG-Bergamo",
		},
		Spec: bsdom.BlockStorageSpec{
			SizeGB:         100,
			SourceImageRef: &commondomain.Reference{Region: "eu-de", Tenant: "tenant-123"},
		},
		Status: &bsdom.BlockStorageStatus{Status: commondomain.Status{State: commondomain.ResourceStateActive}},
	}
}

func TestBlockStorageConverter_FromSECAToAruba(t *testing.T) {
	bs, err := converter.NewBlockStorageConverter().FromSECAToAruba(secaBlockStorage())
	require.NoError(t, err)

	require.Equal(t, "my-block-storage", bs.Name)
	require.Equal(t, "499361fe6f0e4b318e6dc9723bc08427efa461d669f97f79d6486d30", bs.Namespace)
	require.Equal(t, "test-tenant", bs.Spec.Tenant)
	require.Equal(t, int32(100), bs.Spec.SizeGB)
	require.Equal(t, "test-workspace", bs.Spec.ProjectReference.Name)
	require.Equal(t, "ITBG-Bergamo", bs.Spec.Region)
}

// A volume with no region cannot be placed: Aruba resolves both the zone and the size catalog
// within one. Defaulting would provision it somewhere nobody asked for, so the conversion fails.
func TestBlockStorageConverter_missingRegionIsRejected(t *testing.T) {
	bs := secaBlockStorage()
	bs.Region = ""

	_, err := converter.NewBlockStorageConverter().FromSECAToAruba(bs)
	require.ErrorContains(t, err, "region is missing")
	require.ErrorIs(t, err, kernel.ErrValidation)
}

func TestBlockStorageConverter_FromArubaToSECA(t *testing.T) {
	tests := []struct {
		name   string
		input  *v1alpha1.BlockStorage
		assert func(t *testing.T, project *bsdom.BlockStorage)
	}{
		{
			name: "happy path",
			input: &v1alpha1.BlockStorage{
				ObjectMeta: v1.ObjectMeta{
					Name:      "my-block-storage",
					Namespace: "default",
				},
				Spec: v1alpha1.BlockStorageSpec{
					SizeGB:        50,
					Tenant:        "tenant-456",
					Zone:          "IT-BG1",
					BillingPeriod: "Monthly",
					ProjectReference: v1alpha1.ResourceReference{
						Name:      "project-789",
						Namespace: "default",
					},
					Region: "eu-fr",
				},
			},
			assert: func(t *testing.T, bs *bsdom.BlockStorage) {
				t.Helper()
				if bs.Name != "my-block-storage" {
					t.Errorf("expected block storage name 'my-block-storage', got %s", bs.Name)
				}

				if bs.Spec.SizeGB != 50 {
					t.Errorf("expected size 50, got %d", bs.Spec.SizeGB)
				}

				if bs.Spec.SourceImageRef.Tenant != "tenant-456" {
					t.Errorf("expected tenant 'tenant-456', got %s", bs.Spec.SourceImageRef.Tenant)
				}

				if bs.Spec.SourceImageRef.Region != "eu-fr" {
					t.Errorf("expected region 'eu-fr', got %s", bs.Spec.SourceImageRef.Region)
				}

				if bs.Spec.SourceImageRef.Workspace != "project-789" {
					t.Errorf("expected workspace 'project-789', got %s", bs.Spec.SourceImageRef.Workspace)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := &converter.BlockStorageConverter{}
			result, err := converter.FromArubaToSECA(tt.input)
			if err != nil {
				tt.assert(t, nil)
			}

			tt.assert(t, result)
		})
	}
}
