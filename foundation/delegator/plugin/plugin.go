package plugin

import "context"

// ResourcePlugin defines the interface plugins must implement to be loaded by the delegator.
type ResourcePlugin interface {
	// Name returns a unique name for the plugin/provider.
	Name() string
	// Init initializes the plugin and prepares any background resources.
	Init(ctx context.Context) error
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
