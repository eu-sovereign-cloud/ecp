package regional

// StorageSKUDomain represents the domain model for a storage SKU.
type StorageSKUDomain struct {
	Metadata
	Spec StorageSKUSpecDomain
}

// StorageSKUSpecDomain defines the specification for a storage SKU.
type StorageSKUSpecDomain struct {
	Iops          int64
	MinVolumeSize int64
	Type          string
}
