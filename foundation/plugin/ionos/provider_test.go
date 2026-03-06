package ionos

import (
	"context"
	"testing"
)

func TestProvider_Name(t *testing.T) {
	p := &Provider{}
	if got := p.Name(); got != "ionoscloud" {
		t.Errorf("expected Name() = %q, got %q", "ionoscloud", got)
	}
}

func TestProvider_Init(t *testing.T) {
	p := &Provider{}
	if err := p.Init(context.Background()); err != nil {
		t.Errorf("expected Init() to return nil, got %v", err)
	}
}
