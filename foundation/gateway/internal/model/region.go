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
