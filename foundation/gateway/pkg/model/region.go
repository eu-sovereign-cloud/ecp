package model

type RegionDomain struct {
	Meta      Metadata
	Providers []Provider
	Zones     []string
}

func (r RegionDomain) GetName() string {
	return r.Meta.Name
}

func (r RegionDomain) GetNamespace() string {
	return ""
}

func (r RegionDomain) SetName(name string) {
	r.Meta.Name = name
}

func (r RegionDomain) SetNamespace(namespace string) {
	// Regions do not have namespaces; no-op
}

type Provider struct {
	Name    string
	URL     string
	Version string
}
