package v1

import "github.com/eu-sovereign-cloud/ecp/apis/regional"

type StorageSpec struct {

	// Size of the storage in gigabytes
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10000
	SizeGB int64 `json:"sizeGB"`
	// SKU of the storage
	Sku string `json:"sku,omitempty"`

	regional.CommonSpec `json:",inline"`
}
