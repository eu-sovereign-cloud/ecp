package regional

// ImageDomain represents the domain model for an image.
type ImageDomain struct {
	Metadata
	Spec   ImageSpecDomain
	Status *ImageStatusDomain
}

// ImageSpecDomain defines the specification for an image.
type ImageSpecDomain struct {
	BlockStorageRef ReferenceDomain
	CpuArchitecture string
	Initializer     string
	Boot            string
}

// ImageStatusDomain defines the status for an image.
type ImageStatusDomain struct {
	SizeMB *int
	StatusDomain
}
