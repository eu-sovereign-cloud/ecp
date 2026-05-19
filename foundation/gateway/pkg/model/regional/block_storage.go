package regional

// BlockStorageDomain represents the domain model for a block storage instance.
type BlockStorageDomain struct {
	Metadata
	Spec   BlockStorageSpecDomain
	Status *BlockStorageStatusDomain
}

// BlockStorageSpecDomain defines the specification for a block storage instance.
type BlockStorageSpecDomain struct {
	SizeGB         int
	SkuRef         ReferenceDomain
	SourceImageRef *ReferenceDomain
}

// BlockStorageStatusDomain defines the status for a block storage instance.
type BlockStorageStatusDomain struct {
	AttachedTo *ReferenceDomain
	SizeGB     int
	StatusDomain
}
