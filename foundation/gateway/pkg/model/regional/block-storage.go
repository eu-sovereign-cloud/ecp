package regional

// BlockStorageDomain represents the domain model for a block storage instance.
type BlockStorageDomain struct {
	Metadata
	Spec   BlockStorageSpec
	Status *BlockStorageStatus
}

// BlockStorageSpec defines the specification for a block storage instance.
type BlockStorageSpec struct {
	SizeGB         int
	SkuRef         ReferenceObject
	SourceImageRef *ReferenceObject
}

// BlockStorageStatus defines the status for a block storage instance.
type BlockStorageStatus struct {
	AttachedTo *ReferenceObject
	Conditions []StatusConditionDomain
	SizeGB     int
	State      *ResourceStateDomain
}
