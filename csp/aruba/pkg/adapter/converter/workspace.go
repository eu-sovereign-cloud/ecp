package converter

import (
	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

type WorkspaceProjectConverter struct{}

func NewWorkspaceProjectConverter() *WorkspaceProjectConverter {
	return &WorkspaceProjectConverter{}
}

func (c *WorkspaceProjectConverter) FromSECAToAruba(from *wsdom.Workspace) (*v1alpha1.Project, error) {
	spec := v1alpha1.ProjectSpec{}

	if v, ok := from.Spec["description"].(string); ok {
		spec.Description = v
	}

	namespace := k8sadapter.ComputeNamespace(&res.Scope{
		Tenant: from.GetTenant(),
	})

	if v, ok := from.Spec["tags"].([]string); ok {
		spec.Tags = v
	} else if v, ok := from.Spec["tags"].([]any); ok {
		for _, t := range v {
			if s, ok := t.(string); ok {
				spec.Tags = append(spec.Tags, s)
			}
		}
	}

	spec.Tenant = from.GetTenant()

	return &v1alpha1.Project{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Project",
			APIVersion: "arubacloud.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      from.GetName(),
			Namespace: namespace,
			Labels: map[string]string{
				"seca.workspace/workspace": from.GetWorkspace(),
				"seca.workspace/tenant":    from.GetTenant(),
				"seca.workspace/namespace": namespace,
			},
		},
		Spec:   spec,
		Status: v1alpha1.ResourceStatus{},
	}, nil
}

func (c *WorkspaceProjectConverter) FromArubaToSECA(
	from *v1alpha1.Project,
) (*wsdom.Workspace, error) {
	spec := wsdom.WorkspaceSpec{
		"description": from.Spec.Description,
		"tenant":      from.Spec.Tenant,
		"tags":        from.Spec.Tags,
	}

	tenant := from.Labels["seca.workspace/tenant"]
	if tenant == "" {
		tenant = from.Spec.Tenant
	}

	return &wsdom.Workspace{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{
				Name: from.Name,
			},
			Scope: res.Scope{
				Tenant: tenant,
			},
		},
		Spec: spec,
		Status: &wsdom.WorkspaceStatus{
			Status: commondomain.Status{},
		},
	}, nil
}
