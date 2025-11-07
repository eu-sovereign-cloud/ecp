package plugin

import (
	"context"
	"errors"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PluginResult captures the outcome of a reconciliation attempt.
type PluginResult struct {
	ExternalID     string
	State          string // Pending, InProgress, Succeeded, Failed
	Message        string
	RequeueAfter   time.Duration // 0 means no requeue suggestion
	TransientError bool          // true if retry is advised
}

// ResourcePlugin defines the interface plugins must implement to be loaded by the delegator.
type ResourcePlugin interface {
	// Name returns a unique name for the plugin/provider.
	Name() string
	// Init initializes the plugin and prepares any background resources.
	Init(ctx context.Context) error
	// SupportedKinds returns the GVK strings this plugin can handle (e.g. storage.v1.secapi.cloud/Storage)
	SupportedKinds() []string
	// Reconcile executes create/update logic for the given object.
	Reconcile(ctx context.Context, obj client.Object) (PluginResult, error)
	// Delete executes deletion logic for the given object.
	Delete(ctx context.Context, obj client.Object) error
}

// Registry maintains a list of registered plugins.
var Registry = make(map[string]ResourcePlugin)

// Register adds the plugin to the registry. Panics if name is duplicate.
func Register(p ResourcePlugin) {
	name := p.Name()
	if _, exists := Registry[name]; exists {
		panic("plugin already registered: " + name)
	}
	Registry[name] = p
}

// ErrNotHandled indicates the plugin does not support the object's kind.
var ErrNotHandled = errors.New("plugin does not handle this kind")
