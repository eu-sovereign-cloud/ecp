package regional

// StorageSKUDomain represents the domain model for a storage SKU.
type StorageSKUDomain struct {
	Metadata
	Spec StorageSKUSpec
}

// StorageSKUSpec defines the specification for a storage SKU.
type StorageSKUSpec struct {
	Iops          int64
	MinVolumeSize int64
	Type          string
}
