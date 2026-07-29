package metrics

import (
	"context"
	"time"

	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
)

// InstrumentedChecker wraps an authzport.Checker and records the duration of
// each Authorize call into the ecp_gateway_authz_check_duration_seconds histogram
// (metric b). The impl label identifies the underlying implementation ("direct"
// or "cached"), matching the label value used in ObserveRBACFetch.
//
// If the inner checker implements the optional starter interface
// (i.e. it is a CachedChecker with a Start method), InstrumentedChecker forwards
// Start so that auth.StartChecker's type-assertion still resolves after wrapping.
type InstrumentedChecker struct {
	inner authzport.Checker
	impl  string
}

// NewInstrumentedChecker returns an InstrumentedChecker that wraps inner and
// labels observations with impl ("direct" or "cached").
func NewInstrumentedChecker(inner authzport.Checker, impl string) *InstrumentedChecker {
	return &InstrumentedChecker{inner: inner, impl: impl}
}

// Authorize implements authzport.Checker; times the call and records it.
func (c *InstrumentedChecker) Authorize(ctx context.Context, claim authzport.AuthorizationClaim) (authzport.Decision, error) {
	start := time.Now()
	dec, err := c.inner.Authorize(ctx, claim)
	ObserveAuthzCheck(c.impl, time.Since(start))
	return dec, err
}

// Start delegates to the inner checker when it exposes a Start method.
// This preserves the lifecycle contract of CachedChecker after wrapping,
// so auth.StartChecker's type-assertion continues to work.
func (c *InstrumentedChecker) Start(ctx context.Context) error {
	type starter interface{ Start(context.Context) error }
	if s, ok := c.inner.(starter); ok {
		return s.Start(ctx)
	}
	return nil
}
