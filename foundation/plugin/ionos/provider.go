package ionos

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storagev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/block-storage/storages/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/plugin"
)

// Provider implements the delegator ResourcePlugin interface for IONOS, translating Storage to Crossplane XStorage.
type Provider struct {
	client client.Client
}

func (p *Provider) Name() string                   { return "ionoscloud" }
func (p *Provider) Init(ctx context.Context) error { return nil }
func (p *Provider) SupportedKinds() []string       { return []string{"storage.v1.secapi.cloud/Storage"} }
func (p *Provider) SetClient(c client.Client)      { p.client = c }

// todo replace with types from composite ionos provider when available
var xStorageGVK = schema.GroupVersionKind{Group: "xplatform.seca.crossplane.io", Version: "v1alpha1", Kind: "XStorage"}

func (p *Provider) Reconcile(ctx context.Context, obj client.Object) (plugin.PluginResult, error) {
	// Type assert to Storage
	sto, ok := obj.(*storagev1.Storage)
	if !ok {
		return plugin.PluginResult{State: "Failed", Message: "unsupported object type"}, nil
	}

	ann := obj.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	stateKey := "plugin.ionos/state"
	state := ann[stateKey]
	name := fmt.Sprintf("xsto-%s", obj.GetName())

	// Build desired XR spec by copying BlockStorageSpec fields
	spec := map[string]any{
		"sizeGB": sto.Spec.SizeGB,
		"skuRef": sto.Spec.SkuRef,
	}
	if sto.Spec.SourceImageRef != nil {
		spec["sourceImageRef"] = sto.Spec.SourceImageRef
	}

	if state == "" { // create XR
		xr := &unstructured.Unstructured{}
		xr.SetGroupVersionKind(xStorageGVK)
		xr.SetNamespace(obj.GetNamespace())
		xr.SetName(name)
		xr.Object["spec"] = spec
		// Add owner reference for garbage collection (best effort; skipping deep details)
		// NOTE: In production, use controllerutil.SetOwnerReference
		if err := p.client.Create(ctx, xr); err != nil {
			return plugin.PluginResult{State: "Failed", Message: "xr create error: " + err.Error()}, err
		}
		ann[stateKey] = "InProgress"
		obj.SetAnnotations(ann)
		return plugin.PluginResult{State: "InProgress", Message: "xr created", RequeueAfter: 3 * 1e9}, nil
	}
	if state == "InProgress" { // simulate becoming ready by checking a dummy condition
		// Would normally fetch XR and inspect status conditions
		ann[stateKey] = "Succeeded"
		ann["plugin.ionos/id"] = name
		obj.SetAnnotations(ann)
		return plugin.PluginResult{State: "Succeeded", Message: "xr ready", ExternalID: name}, nil
	}
	return plugin.PluginResult{State: "Succeeded", ExternalID: ann["plugin.ionos/id"], Message: "already ready"}, nil
}

func (p *Provider) Delete(ctx context.Context, obj client.Object) error {
	ann := obj.GetAnnotations()
	if ann == nil {
		return nil
	}
	name := ann["plugin.ionos/id"]
	if name == "" {
		return nil
	}
	xr := &unstructured.Unstructured{}
	xr.SetGroupVersionKind(xStorageGVK)
	xr.SetNamespace(obj.GetNamespace())
	xr.SetName(name)
	_ = p.client.Delete(ctx, xr) // ignore not found
	ann["plugin.ionos/deleted"] = "true"
	obj.SetAnnotations(ann)
	return nil
}

func init() { plugin.Register(&Provider{}) }
