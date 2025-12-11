package ionos

import (
	"context"
)

// Provider implements the delegator ResourcePlugin interface for IONOS.
type Provider struct{}

func (p *Provider) Name() string { return "ionoscloud" }

func (p *Provider) Init(ctx context.Context) error {
	// TODO: initialization logic
	return nil
}
