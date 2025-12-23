package converter

import (
	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type WorkspaceProjectConverter struct {
	Namespace string
}

func (c *WorkspaceProjectConverter) FromSECAToAruba(from *regional.WorkspaceDomain) (*v1alpha1.Project, error) {
	spec := v1alpha1.ProjectSpec{}

	if v, ok := from.Spec["description"].(string); ok {
		spec.Description = v
	}

	spec.Tenant = from.Metadata.Tenant

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

	project := &v1alpha1.Project{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Project",
			APIVersion: "arubacloud.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      from.Metadata.Workspace,
			Namespace: c.Namespace,
			Labels: map[string]string{
				"seca.workspace/id": from.Metadata.Workspace,
			},
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

	ws := &regional.WorkspaceDomain{
		Metadata: regional.Metadata{
			Tenant:    from.Spec.Tenant,
			Workspace: from.Name,
		},
		Spec: spec,
		Status: regional.WorkspaceStatusDomain{
			StatusDomain: regional.StatusDomain{},
		},
	}

	return ws, nil
}
