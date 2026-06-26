package kubernetes

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	k8slabels "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/labels"
	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"

	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
)

// ImageFromCR converts either a concrete *Image or *unstructured.Unstructured
// into a *imgdom.Image.
func ImageFromCR(obj client.Object) (*imgdom.Image, error) {
	var cr Image

	switch t := obj.(type) {
	case *Image:
		cr = *t
	case *unstructured.Unstructured:
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(t.Object, &cr); err != nil {
			return nil, fmt.Errorf("failed to convert unstructured to Image: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported object type %T", obj)
	}

	crLabels := cr.GetLabels()
	internalLabels := k8slabels.GetInternalLabels(crLabels)
	keyedLabels := k8slabels.GetKeyedLabels(crLabels)

	img := &imgdom.Image{
		Spec: imgdom.ImageSpec{
			BlockStorageRef: commonbackend.ReferenceFromCR(cr.Spec.BlockStorageRef),
			CpuArchitecture: string(cr.Spec.CpuArchitecture),
			Boot:            string(cr.Spec.Boot),
			Initializer:     string(cr.Spec.Initializer),
		},
	}
	img.Name = cr.GetName()
	img.ResourceVersion = cr.GetResourceVersion()
	img.CreatedAt = cr.GetCreationTimestamp().Time
	img.UpdatedAt = cr.GetCreationTimestamp().Time
	img.Provider = strings.ReplaceAll(internalLabels[k8slabels.InternalProviderLabel], "_", "/")
	img.Tenant = internalLabels[k8slabels.InternalTenantLabel]
	img.Region = internalLabels[k8slabels.InternalRegionLabel]
	img.Labels = k8slabels.KeyedToOriginal(keyedLabels, cr.CommonData.Labels)
	img.Annotations = cr.CommonData.Annotations
	img.Extensions = cr.CommonData.Extensions

	if ts := cr.GetDeletionTimestamp(); ts != nil {
		img.DeletedAt = &ts.Time
	}

	img.Status = &imgdom.ImageStatus{}
	if cr.Status != nil {
		img.Status = &imgdom.ImageStatus{
			SizeMB: cr.Status.SizeMB,
		}
		img.Status.State = commonbackend.ResourceStateFromCR(cr.Status.State)
		img.Status.Conditions = commonbackend.ConditionsFromCR(cr.Status.Conditions)
	} else {
		img.Status.PushCondition(commondomain.DefaultPendingCondition)
	}

	return img, nil
}

// ImageToCR converts a *imgdom.Image to a Kubernetes Image CR.
func ImageToCR(img *imgdom.Image) (client.Object, error) {
	if img == nil {
		return nil, fmt.Errorf("image is nil")
	}

	crLabels := k8slabels.OriginalToKeyed(img.Labels)
	crLabels[k8slabels.InternalTenantLabel] = img.Tenant
	crLabels[k8slabels.InternalProviderLabel] = strings.ReplaceAll(img.Provider, "/", "_")
	crLabels[k8slabels.InternalRegionLabel] = img.Region

	cr := &Image{
		ObjectMeta: v1.ObjectMeta{
			Name:            img.Name,
			Namespace:       k8sadapter.ComputeNamespace(tenantOnlyScope(img.Tenant)),
			Labels:          crLabels,
			ResourceVersion: img.ResourceVersion,
		},
		CommonData: schemav1.CommonData{
			Annotations: img.Annotations,
			Extensions:  img.Extensions,
			Labels:      slices.Collect(maps.Keys(img.Labels)),
		},
		Spec: ImageSpec{
			BlockStorageRef: commonbackend.ReferenceToCR(img.Spec.BlockStorageRef),
			CpuArchitecture: ImageSpecCpuArchitecture(img.Spec.CpuArchitecture),
			Boot:            ImageSpecBoot(img.Spec.Boot),
			Initializer:     ImageSpecInitializer(img.Spec.Initializer),
		},
	}
	cr.SetGroupVersionKind(ImageGVK)

	if img.Status != nil && len(img.Status.Conditions) > 0 {
		state := commonbackend.ResourceStateToCR(img.Status.State)
		if state == nil {
			return nil, fmt.Errorf("failed to convert resource state to CR")
		}
		cr.Status = &ImageStatus{
			SizeMB:     img.Status.SizeMB,
			Conditions: commonbackend.ConditionsToCR(img.Status.Conditions),
			State:      *state,
		}
	}

	return cr, nil
}

// tenantOnlyScope returns a scope with only the tenant set.
// Image CRs live in the tenant namespace (images are tenant-scoped, not workspace-scoped).
func tenantOnlyScope(tenant string) *kernelresource.Scope {
	return &kernelresource.Scope{Tenant: tenant}
}
