package converter_test

import (
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"

	"github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/adapter/converter"
)

func TestWorkspaceProjectConverter_FromSECAToAruba(t *testing.T) {
	tests := []struct {
		name      string
		input     *regional.WorkspaceDomain
		namespace string
		assert    func(t *testing.T, project *v1alpha1.Project)
	}{
		{
			name:      "happy path with description tags and default",
			namespace: "test-namespace",
			input: &regional.WorkspaceDomain{
				Metadata: regional.Metadata{
					Region: "region-1",
					CommonMetadata: model.CommonMetadata{
						Name: "workspace-abc",
					},
					Scope: scope.Scope{
						Tenant:    "tenant-123",
						Workspace: "workspace-abc",
					},
				},
				Spec: map[string]interface{}{
					"description": "My test project",
					"tags":        []interface{}{"tag1", "tag2"},
					"default":     true,
				},
			},
			assert: func(t *testing.T, project *v1alpha1.Project) {
				t.Helper()

				if project.Name != "workspace-abc" {
					t.Errorf("expected project name 'workspace-abc', got %s", project.Name)
				}

				if project.Namespace != "test-namespace" {
					t.Errorf("expected namespace 'test-namespace', got %s", project.Namespace)
				}

				if project.Spec.Tenant != "tenant-123" {
					t.Errorf("expected tenant 'tenant-123', got %s", project.Spec.Tenant)
				}

				if project.Spec.Description != "My test project" {
					t.Errorf(
						"expected description 'My test project', got %s",
						project.Spec.Description,
					)
				}

				if len(project.Spec.Tags) != 2 ||
					project.Spec.Tags[0] != "tag1" ||
					project.Spec.Tags[1] != "tag2" {
					t.Errorf(
						"expected tags ['tag1', 'tag2'], got %v",
						project.Spec.Tags,
					)
				}

				if !project.Spec.Default {
					t.Errorf("expected default=true, got false")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := &converter.WorkspaceProjectConverter{
				Namespace: tt.namespace,
			}

			project, err := converter.FromSECAToAruba(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			tt.assert(t, project)
		})
	}
}

func TestWorkspaceProjectConverter_FromArubaToSECA(t *testing.T) {
	tests := []struct {
		name      string
		input     *v1alpha1.Project
		namespace string
		assert    func(t *testing.T, workspace *regional.WorkspaceDomain)
	}{
		{
			name:      "happy path with description tags and default",
			namespace: "test-namespace",
			input: &v1alpha1.Project{
				Spec: v1alpha1.ProjectSpec{
					Tenant:      "tenant-123",
					Description: "My test project",
					Tags:        []string{"tag1", "tag2"},
					Default:     true,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "workspace-abc",
				},
			},
			assert: func(t *testing.T, workspace *regional.WorkspaceDomain) {
				t.Helper()

				if workspace.Workspace != "workspace-abc" {
					t.Errorf("expected workspace name 'workspace-abc', got %s", workspace.Workspace)
				}

				if workspace.Tenant != "tenant-123" {
					t.Errorf("expected tenant 'tenant-123', got %s", workspace.Tenant)
				}

				if desc, ok := workspace.Spec["description"].(string); !ok || desc != "My test project" {
					t.Errorf(
						"expected description 'My test project', got %v",
						workspace.Spec["description"],
					)
				}

				if tags, ok := workspace.Spec["tags"].([]string); !ok ||
					len(tags) != 2 || tags[0] != "tag1" || tags[1] != "tag2" {
					t.Errorf(
						"expected tags ['tag1', 'tag2'], got %v",
						workspace.Spec["tags"],
					)
				}

				if def, ok := workspace.Spec["default"].(bool); !ok || !def {
					t.Errorf("expected default=true, got %v", workspace.Spec["default"])
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := &converter.WorkspaceProjectConverter{
				Namespace: tt.namespace,
			}

			workspace, err := converter.FromArubaToSECA(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			tt.assert(t, workspace)
		})
	}
}
