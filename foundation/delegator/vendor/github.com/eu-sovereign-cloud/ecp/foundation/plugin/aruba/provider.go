package aruba

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/plugin"
)

// Provider implements the delegator ResourcePlugin interface for Aruba.
type Provider struct{}

func (p *Provider) Name() string { return "aruba" }

func (p *Provider) Init(ctx context.Context) error { return nil }

func (p *Provider) SupportedKinds() []string { return []string{"storage.v1.secapi.cloud/Storage"} }

func (p *Provider) Reconcile(ctx context.Context, obj client.Object) (plugin.PluginResult, error) {
	ann := obj.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	key := "plugin.aruba/state"
	state := ann[key]
	if state == "" { // first call
		ann[key] = "InProgress"
		obj.SetAnnotations(ann)
		return plugin.PluginResult{State: "InProgress", Message: "aruba provisioning started", RequeueAfter: 2_000_000_000}, nil
	}
	if state == "InProgress" { // second call -> succeed
		ann[key] = "Succeeded"
		ann["plugin.aruba/id"] = fmt.Sprintf("arb-%s", obj.GetUID())
		obj.SetAnnotations(ann)
		return plugin.PluginResult{State: "Succeeded", Message: "aruba provisioning complete", ExternalID: ann["plugin.aruba/id"]}, nil
	}
	return plugin.PluginResult{State: "Succeeded", ExternalID: ann["plugin.aruba/id"], Message: "already complete"}, nil
}

func (p *Provider) Delete(ctx context.Context, obj client.Object) error {
	// Simulate delete by marking an annotation
	ann := obj.GetAnnotations()
	if ann == nil {
		return nil
	}
	ann["plugin.aruba/deleted"] = "true"
	obj.SetAnnotations(ann)
	return nil
}

func init() {
	plugin.Register(&Provider{})
}
