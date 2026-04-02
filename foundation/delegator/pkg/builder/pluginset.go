package builder

import "github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"

// PluginSet is a collection of plugins that a specific provider will implement.
type PluginSet struct {
	BlockStorage plugin.BlockStorage
	Workspace    plugin.Workspace
}

// PluginSetOption is a function that configures the PluginSet.
type PluginSetOption func(*PluginSet)

// NewPluginSet creates a new PluginSet with the provided options.
func NewPluginSet(opts ...PluginSetOption) *PluginSet {
	ps := &PluginSet{}
	for _, opt := range opts {
		opt(ps)
	}
	return ps
}

// WithBlockStorage sets the BlockStorage plugin for the PluginSet.
func WithBlockStorage(p plugin.BlockStorage) PluginSetOption {
	return func(ps *PluginSet) {
		ps.BlockStorage = p
	}
}

// WithWorkspace sets the Workspace plugin for the PluginSet.
func WithWorkspace(p plugin.Workspace) PluginSetOption {
	return func(ps *PluginSet) {
		ps.Workspace = p
	}
}
