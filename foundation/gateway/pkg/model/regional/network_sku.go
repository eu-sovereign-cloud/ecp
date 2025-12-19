package regional

type NetworkSKUDomain struct {
	Metadata
	Spec NetworkSKUSpec
}

type NetworkSKUSpec struct {
	Bandwidth int
	Packets   int
}
