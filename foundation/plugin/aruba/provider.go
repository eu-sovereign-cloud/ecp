package aruba

import (
	"context"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/plugin"
)

// Provider implements the delegator ResourcePlugin interface for Aruba.
type Provider struct{}

func (p *Provider) Name() string { return "aruba" }

func (p *Provider) Init(ctx context.Context) error {
	// TODO: initialization logic
	return nil
}

func init() {
	plugin.Register(&Provider{})
}
