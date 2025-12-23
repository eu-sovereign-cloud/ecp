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

// KeyPairSpec defines the desired state of KeyPair.
type KeyPairSpec struct {
	// Tenant is the owning account/tenant of this keypair
	Tenant string `json:"tenant,omitempty"`

	// Tags are labels associated with the keypair
	// +kubebuilder:validation:Optional
	Tags []string `json:"tags,omitempty"`

	// Location specifies the location for the keypair
	// +kubebuilder:validation:Required
	Location Location `json:"location"`

	// Value specifies the SSH public key value
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Value string `json:"value"`

	// ProjectReference references the Project that owns this keypair
	// +kubebuilder:validation:Required
	ProjectReference ResourceReference `json:"projectReference"`
}

// KeyPairStatus defines the observed state of KeyPair.
type KeyPairStatus struct {
	ResourceStatus `json:",inline"`

	// ProjectID is the project ID where this keypair is created
	// +kubebuilder:validation:Optional
	ProjectID string `json:"projectID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=kp
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Resource ID",type="string",JSONPath=".status.resourceID"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// KeyPair is the Schema for the keypairs API.
type KeyPair struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KeyPairSpec   `json:"spec,omitempty"`
	Status KeyPairStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// KeyPairList contains a list of KeyPair.
type KeyPairList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KeyPair `json:"items"`
}

func init() {
	SchemeBuilder.Register(&KeyPair{}, &KeyPairList{})
}
