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

// SecurityGroupSpec defines the desired state of SecurityGroup.
type SecurityGroupSpec struct {
	// Tenant is the owning account/tenant of this security group
	Tenant string `json:"tenant,omitempty"`

	// Tags are labels associated with the security group
	// +kubebuilder:validation:Optional
	Tags []string `json:"tags,omitempty"`

	// Location specifies the location for the security group
	// +kubebuilder:validation:Required
	Location Location `json:"location"`

	// Default indicates whether this is a default security group
	// +kubebuilder:validation:Optional
	Default bool `json:"default,omitempty"`

	// VpcReference references the ArubaVpc that owns this security group
	// +kubebuilder:validation:Required
	VpcReference ResourceReference `json:"vpcReference"`

	// ProjectReference references the Project that owns this security group
	// +kubebuilder:validation:Required
	ProjectReference ResourceReference `json:"projectReference"`
}

// SecurityGroupStatus defines the observed state of SecurityGroup.
type SecurityGroupStatus struct {
	ResourceStatus `json:",inline"`

	// ProjectID is the project ID where this security group is created
	// +kubebuilder:validation:Optional
	ProjectID string `json:"projectID,omitempty"`

	// VpcID is the VPC ID where this security group is created
	// +kubebuilder:validation:Optional
	VpcID string `json:"vpcID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=sg
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Resource ID",type="string",JSONPath=".status.resourceID"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// SecurityGroup is the Schema for the securitygroups API.
type SecurityGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecurityGroupSpec   `json:"spec,omitempty"`
	Status SecurityGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecurityGroupList contains a list of SecurityGroup.
type SecurityGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecurityGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecurityGroup{}, &SecurityGroupList{})
}
