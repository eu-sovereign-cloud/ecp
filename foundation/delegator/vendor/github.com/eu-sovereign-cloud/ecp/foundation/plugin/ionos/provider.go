package ionos

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/plugin"
)

// Provider implements the delegator ResourcePlugin interface for IONOS.
type Provider struct{}

func (p *Provider) Name() string { return "ionoscloud" }

func (p *Provider) Init(ctx context.Context) error {
	// TODO: initialization logic
	return nil
}

func (p *Provider) SupportedKinds() []string { return []string{"storage.v1.secapi.cloud/Storage"} }

func (p *Provider) Reconcile(ctx context.Context, obj client.Object) (plugin.PluginResult, error) {
	ann := obj.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	key := "plugin.ionos/state"
	state := ann[key]
	if state == "" {
		ann[key] = "InProgress"
		obj.SetAnnotations(ann)
		return plugin.PluginResult{State: "InProgress", Message: "ionos provisioning started", RequeueAfter: 2_000_000_000}, nil
	}
	if state == "InProgress" {
		ann[key] = "Succeeded"
		ann["plugin.ionos/id"] = fmt.Sprintf("ion-%s", obj.GetUID())
		obj.SetAnnotations(ann)
		return plugin.PluginResult{State: "Succeeded", ExternalID: ann["plugin.ionos/id"], Message: "ionos provisioning complete"}, nil
	}
	return plugin.PluginResult{State: "Succeeded", ExternalID: ann["plugin.ionos/id"], Message: "already complete"}, nil
}

func (p *Provider) Delete(ctx context.Context, obj client.Object) error {
	ann := obj.GetAnnotations()
	if ann == nil {
		return nil
	}
	ann["plugin.ionos/deleted"] = "true"
	obj.SetAnnotations(ann)
	return nil
}

func init() {
	plugin.Register(&Provider{})
}
