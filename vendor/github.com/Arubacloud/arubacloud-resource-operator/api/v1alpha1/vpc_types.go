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

// VpcSpec defines the desired state of Vpc.
type VpcSpec struct {
	// Tenant is the owning account/tenant of this vpc
	Tenant string `json:"tenant,omitempty"`

	// Tags are labels associated with the vpc
	// +kubebuilder:validation:Optional
	Tags []string `json:"tags,omitempty"`

	// Location specifies the location for the vpc
	// +kubebuilder:validation:Required
	Location Location `json:"location"`

	// ProjectReference references the Project that owns this vpc
	// +kubebuilder:validation:Required
	ProjectReference ResourceReference `json:"projectReference"`
}

// VpcStatus defines the observed state of Vpc.
type VpcStatus struct {
	ResourceStatus `json:",inline"`

	// ProjectID is the project ID where this vpc is created
	// +kubebuilder:validation:Optional
	ProjectID string `json:"projectID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=vpc
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Resource ID",type="string",JSONPath=".status.resourceID"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Vpc is the Schema for the vpcs API.
type Vpc struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VpcSpec   `json:"spec,omitempty"`
	Status VpcStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// VpcList contains a list of Vpc.
type VpcList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Vpc `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Vpc{}, &VpcList{})
}
