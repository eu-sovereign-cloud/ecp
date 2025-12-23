package converter

import (
	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

type WorkspaceProjectConverter struct {
}

func NewWorkspaceProjectConverter() *WorkspaceProjectConverter {
	return &WorkspaceProjectConverter{}
}

func (c *WorkspaceProjectConverter) FromSECAToAruba(from *regional.WorkspaceDomain) (*v1alpha1.Project, error) {
	spec := v1alpha1.ProjectSpec{}

	if v, ok := from.Spec["description"].(string); ok {
		spec.Description = v
	}

	// namespace := kubernetesadapter.ComputeNamespace(&from.Metadata.Scope)

	namespace := kubernetesadapter.ComputeNamespace(&scope.Scope{
		Tenant: from.Metadata.Tenant,
	})

	if v, ok := from.Spec["tags"].([]string); ok {
		spec.Tags = v
	} else if v, ok := from.Spec["tags"].([]interface{}); ok {
		for _, t := range v {
			if s, ok := t.(string); ok {
				spec.Tags = append(spec.Tags, s)
			}
		}
	}

	if v, ok := from.Spec["default"].(bool); ok {
		spec.Default = v
	}

	spec.Tenant = from.Scope.Tenant

	project := &v1alpha1.Project{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Project",
			APIVersion: "arubacloud.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      from.Metadata.Name,
			Namespace: namespace,
			Labels: map[string]string{
				// "seca.workspace/workspace": from.Metadata.Workspace,
				"seca.workspace/tenant":    from.Scope.Tenant,
				"seca.workspace/namespace": namespace},
		},
		Spec:   spec,
		Status: v1alpha1.ResourceStatus{},
	}

	return project, nil
}

func (c *WorkspaceProjectConverter) FromArubaToSECA(
	from *v1alpha1.Project,
) (*regional.WorkspaceDomain, error) {

	spec := regional.WorkspaceSpec{
		"description": from.Spec.Description,
		"tenant":      from.Spec.Tenant,
		"tags":        from.Spec.Tags,
		"default":     from.Spec.Default,
	}

	tenant := from.Labels["seca.workspace/tenant"]

	if tenant == "" {
		tenant = from.Spec.Tenant
	}

	ws := &regional.WorkspaceDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{
				Name: from.Name,
			},
			Scope: scope.Scope{
				Tenant: tenant,
			},
		},
		Spec: spec,
		Status: &regional.WorkspaceStatusDomain{
			StatusDomain: regional.StatusDomain{},
		},
	}

	return ws, nil
}
