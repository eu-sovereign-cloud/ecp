package regional

type NetworkSKUDomain struct {
	Metadata
	Spec NetworkSKUSpecDomain
}

type NetworkSKUSpecDomain struct {
	Bandwidth int
	Packets   int
}
