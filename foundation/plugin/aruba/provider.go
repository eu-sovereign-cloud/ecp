package aruba

import (
	"context"
)

// Provider implements the delegator ResourcePlugin interface for Aruba.
type Provider struct{}

func (p *Provider) Name() string { return "aruba" }

func (p *Provider) Init(ctx context.Context) error {
	// TODO: initialization logic
	return nil
}
