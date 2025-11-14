package model

type (
	Zone         string
	RegionDomain struct {
		Metadata
		Providers []Provider
		Zones     []Zone
	}
)

type Provider struct {
	Name    string
	URL     string
	Version string
}
