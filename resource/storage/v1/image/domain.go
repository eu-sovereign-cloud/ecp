// Package image defines the image resource domain model and identity constants.
package image

import "github.com/eu-sovereign-cloud/ecp/resource/common/domain"

// Identity constants for the image resource.
const (
	Kind       = "Image"
	Resource   = "images"
	Group      = "storage.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.storage/v1"
)

// Image represents the domain model for an image.
type Image struct {
	domain.RegionalMetadata
	Spec   ImageSpec
	Status *ImageStatus
}

// ImageSpec defines the specification for an image.
type ImageSpec struct {
	BlockStorageRef domain.Reference
	CpuArchitecture string
	Boot            string
	Initializer     string
}

// ImageStatus defines the status for an image.
type ImageStatus struct {
	SizeMB *int
	domain.Status
}
