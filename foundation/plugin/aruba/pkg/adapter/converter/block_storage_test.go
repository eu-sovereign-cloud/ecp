package converter_test

import (
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"

	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/converter"
)

func TestBlockStorageConverter_FromSECAToAruba(t *testing.T) {
	tests := []struct {
		name   string
		input  *regional.BlockStorageDomain
		assert func(t *testing.T, project *v1alpha1.BlockStorage)
	}{
		{
			name: "happy path",
			input: &regional.BlockStorageDomain{
				Metadata: regional.Metadata{
					Scope: scope.Scope{
						Workspace: "test-workspace",
						Tenant:    "test-tenant",
					},
					CommonMetadata: model.CommonMetadata{
						Name: "my-block-storage",
					},
				},
				Spec: regional.BlockStorageSpecDomain{
					SizeGB: 100,
					SourceImageRef: &regional.ReferenceObjectDomain{
						Region: "eu-de",
						Tenant: "tenant-123",
					},
				},
				Status: &regional.BlockStorageStatusDomain{
					StatusDomain: regional.StatusDomain{
						State: regional.ResourceStateActive,
					},
				},
			},
			assert: func(t *testing.T, bs *v1alpha1.BlockStorage) {
				t.Helper()

				if bs.Name != "my-block-storage" {
					t.Errorf("expected block storage name 'my-block-storage', got %s", bs.Name)
				}
				if bs.Namespace != "499361fe6f0e4b318e6dc9723bc08427efa461d669f97f79d6486d30" {
					t.Errorf("expected namespace 'default', got %s", bs.Namespace)
				}

				if bs.Spec.Tenant != "test-tenant" {
					t.Errorf("expected tenant 'tenant-123', got %s", bs.Spec.Tenant)
				}

				if bs.Spec.SizeGB != 100 {
					t.Errorf("expected size 100, got %d", bs.Spec.SizeGB)
				}

				if bs.Spec.ProjectReference.Name != "test-workspace" {
					t.Errorf("expected workspace 'test-workspace', got %s", bs.Spec.ProjectReference.Name)
				}

				if bs.Spec.Region != "eu-de" {
					t.Errorf("expected location 'eu-de', got %s", bs.Spec.Region)
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
			assert: func(t *testing.T, bs *regional.BlockStorageDomain) {
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
