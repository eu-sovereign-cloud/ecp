package model

const (
	RegionBaseURL      = "/providers/seca.region"
	ProviderRegionName = "seca.region/v1"
)

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
