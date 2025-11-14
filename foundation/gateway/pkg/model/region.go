package model

type RegionDomain struct {
	Metadata
	Providers []Provider
	Zones     []string
}

type Provider struct {
	Name    string
	URL     string
	Version string
}
