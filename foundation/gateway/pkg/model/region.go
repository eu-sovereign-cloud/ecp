package model

type (
	ZoneDomain   string
	RegionDomain struct {
		Metadata
		Providers []ProviderDomain
		Zones     []ZoneDomain
	}
)

type ProviderDomain struct {
	Name    string
	URL     string
	Version string
}
