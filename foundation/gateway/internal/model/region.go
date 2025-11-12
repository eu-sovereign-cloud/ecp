package model

type RegionDomain struct {
	Meta      Metadata
	Providers []Provider
	Zones     []string
}

type Provider struct {
	Name    string
	URL     string
	Version string
}

// Invariants:
// - At least one zone
// - Provider names must be non-empty
// - Zone strings must be non-empty
