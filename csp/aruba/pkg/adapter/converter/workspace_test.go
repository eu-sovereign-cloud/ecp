package converter_test

import (
	"testing"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes"
	commondomain "github.com/eu-sovereign-cloud/ecp/resources/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resources/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
)

func TestWorkspaceProjectConverter_FromSECAToAruba(t *testing.T) {
	tests := []struct {
		name   string
		input  *wsdom.Workspace
		assert func(t *testing.T, project *v1alpha1.Project)
	}{
		{
			name: "happy path with description tags and default",
			input: &wsdom.Workspace{
				RegionalMetadata: commondomain.RegionalMetadata{
					Region: "region-1",
					CommonMetadata: commondomain.CommonMetadata{
						Name: "workspace-abc",
					},
					Scope: res.Scope{
						Tenant:    "tenant-123",
						Workspace: "workspace-abc",
					},
				},
				Spec: map[string]any{
					"description": "My test project",
					"tags":        []any{"tag1", "tag2"},
					"default":     true,
				},
			},
			assert: func(t *testing.T, project *v1alpha1.Project) {
				t.Helper()

				if project.Name != "workspace-abc" {
					t.Errorf("expected project name 'workspace-abc', got %s", project.Name)
				}

				if project.Namespace != k8sadapter.ComputeNamespace(&res.Scope{Tenant: "tenant-123"}) {
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := &converter.WorkspaceProjectConverter{}

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
		name   string
		input  *v1alpha1.Project
		assert func(t *testing.T, workspace *wsdom.Workspace)
	}{
		{
			name: "happy path with description tags and default",
			input: &v1alpha1.Project{
				Spec: v1alpha1.ProjectSpec{
					Tenant:      "tenant-123",
					Description: "My test project",
					Tags:        []string{"tag1", "tag2"},
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "workspace-abc",
					Namespace: "random-value-not-considered",
					Labels: map[string]string{
						"seca.workspace/tenant":    "tenant-456",
						"seca.workspace/workspace": "workspace-123",
					},
				},
			},
			assert: func(t *testing.T, workspace *wsdom.Workspace) {
				t.Helper()

				if workspace.GetWorkspace() != "" {
					t.Errorf("expected workspace empty, got %s", workspace.GetWorkspace())
				}

				if workspace.Tenant != "tenant-456" {
					t.Errorf("expected tenant 'tenant-456', got %s", workspace.Tenant)
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			converter := &converter.WorkspaceProjectConverter{}

			workspace, err := converter.FromArubaToSECA(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			tt.assert(t, workspace)
		})
	}
}
