package builder

import (
	"errors"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
)

// PluginSet is a collection of plugins that a specific provider will implement.
type PluginSet struct {
	BlockStorage plugin.BlockStorage
	Workspace    plugin.Workspace
	Network      plugin.Network
}

// Validate checks that all required plugins are set.
func (ps PluginSet) Validate() error {
	if ps.BlockStorage == nil {
		return errors.New("block storage plugin is required")
	}
	if ps.Workspace == nil {
		return errors.New("workspace plugin is required")
	}
	if ps.Network == nil {
		return errors.New("network plugin is required")
	}
	return nil
}
