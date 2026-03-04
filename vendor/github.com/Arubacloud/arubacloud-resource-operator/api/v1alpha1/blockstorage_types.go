/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BlockStorageSpec defines the desired state of BlockStorage.
type BlockStorageSpec struct {
	// Tenant is the owning account/tenant of this block storage
	Tenant string `json:"tenant,omitempty"`

	// Tags are labels associated with the block storage
	// +kubebuilder:validation:Optional
	Tags []string `json:"tags,omitempty"`

	// Location specifies the location for the block storage
	// +kubebuilder:validation:Required
	Location Location `json:"location"`

	// SizeGb specifies the size of the block storage in GB
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=16384
	SizeGb int32 `json:"sizeGb"`

	// BillingPeriod defines the billing period (Hour, Month, etc.)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Hour;Month
	BillingPeriod string `json:"billingPeriod"`

	// DataCenter specifies the data center for the block storage
	// +kubebuilder:validation:Required
	DataCenter string `json:"dataCenter"`

	// Type specifies the type of the block storage
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=Standard;Performance
	Type string `json:"type,omitempty"`

	// Bootable indicates whether the block storage is bootable
	// +kubebuilder:validation:Optional
	Bootable bool `json:"bootable,omitempty"`

	// Image specifies the image ID for the block storage
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`

	// ProjectReference references the Project that owns this block storage
	// +kubebuilder:validation:Required
	ProjectReference ResourceReference `json:"projectReference"`
}

// BlockStorageStatus defines the observed state of BlockStorage.
type BlockStorageStatus struct {
	ResourceStatus `json:",inline"`

	// ProjectID is the project ID where this block storage is created
	// +kubebuilder:validation:Optional
	ProjectID string `json:"projectID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=bs
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Resource ID",type="string",JSONPath=".status.resourceID"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BlockStorage is the Schema for the blockstorages API.
type BlockStorage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BlockStorageSpec   `json:"spec,omitempty"`
	Status BlockStorageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BlockStorageList contains a list of BlockStorage.
type BlockStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BlockStorage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BlockStorage{}, &BlockStorageList{})
}
