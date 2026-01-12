package converter_test

import (
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
		v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/converter"
)

func TestBlockStorageConverter_FromSECAToAruba(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		input     *regional.BlockStorageDomain
		assert    func(t *testing.T, project *v1alpha1.BlockStorage)
	}{
		{
			name:      "happy path",
			namespace: "default",
			input: &regional.BlockStorageDomain{
				Metadata: model.Metadata{
					CommonMetadata: model.CommonMetadata{
						Name: "my-block-storage",
					},
				},
				Spec: regional.BlockStorageSpec{
					SizeGB: 100,
					SourceImageRef: &regional.ReferenceObject{
						Region: "eu-de",
						Tenant: "tenant-123",
					},
				},
				Status: &regional.BlockStorageStatus{
					State: func() *regional.ResourceState {
						s := regional.ResourceStateActive
						return &s
					}(),
				},
			},
			assert: func(t *testing.T, bs *v1alpha1.BlockStorage) {
				t.Helper()

				if bs.Name != "my-block-storage" {
					t.Errorf("expected block storage name 'my-block-storage', got %s", bs.Name)
				}

				if bs.Namespace != "default" {
					t.Errorf("expected namespace 'default', got %s", bs.Namespace)
				}

				if bs.Spec.Tenant != "tenant-123" {
					t.Errorf("expected tenant 'tenant-123', got %s", bs.Spec.Tenant)
				}

				if bs.Spec.SizeGb != 100 {
					t.Errorf("expected size 100, got %d", bs.Spec.SizeGb)
				}

				if bs.Spec.Location.Value != "eu-de" {
					t.Errorf("expected location 'eu-de', got %s", bs.Spec.Location.Value)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := &converter.BlockStorageConverter{}
			result, err := converter.FromSECAToAruba(tt.input)

			if err != nil {
				tt.assert(t, nil)
			}

			tt.assert(t, result)

		})

	}
}

func TestBlockStorageConverter_FromArubaToSECA(t *testing.T) {
	tests := []struct {
		name   string
		input  *v1alpha1.BlockStorage
		assert func(t *testing.T, project *regional.BlockStorageDomain)
	}{
		{
			name: "happy path",
			input: &v1alpha1.BlockStorage{
				ObjectMeta: v1.ObjectMeta{
					Name:      "my-block-storage",
					Namespace: "default",
				},
				Spec: v1alpha1.BlockStorageSpec{
					SizeGb: 50,
					Tenant: "tenant-456",
					DataCenter: "IT-BG1",
					BillingPeriod: "Monthly",
					ProjectReference: v1alpha1.ResourceReference {
						Name: "project-789",
						Namespace: "default",
					},
					Location: v1alpha1.Location{
						Value: "eu-fr",
					},
				},
			},
			assert: func(t *testing.T, bs *regional.BlockStorageDomain) {
				t.Helper()
				if bs.Metadata.Name != "my-block-storage" {
					t.Errorf("expected block storage name 'my-block-storage', got %s", bs.Metadata.Name)
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
