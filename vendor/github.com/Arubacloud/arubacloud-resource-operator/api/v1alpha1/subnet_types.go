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

// SubnetNetwork defines the network configuration for a subnet
type SubnetNetwork struct {
	// Address specifies the network address in CIDR notation
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([0-9]{1,3}\.){3}[0-9]{1,3}\/[0-9]{1,2}$`
	Address string `json:"address"`
}

// SubnetDHCP defines the DHCP configuration for a subnet
type SubnetDHCP struct {
	// Enabled indicates whether DHCP is enabled for this subnet
	// +kubebuilder:validation:Required
	Enabled bool `json:"enabled"`
}

// SubnetSpec defines the desired state of Subnet.
type SubnetSpec struct {
	// Tenant is the owning account/tenant of this subnet
	Tenant string `json:"tenant,omitempty"`

	// Tags are labels associated with the subnet
	// +kubebuilder:validation:Optional
	Tags []string `json:"tags,omitempty"`

	// Type specifies the type of subnet (e.g., "Advanced")
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Advanced;Basic
	Type string `json:"type"`

	// Default indicates whether this is a default subnet
	// +kubebuilder:validation:Optional
	Default bool `json:"default,omitempty"`

	// Network specifies the network configuration
	// +kubebuilder:validation:Required
	Network SubnetNetwork `json:"network"`

	// DHCP specifies the DHCP configuration
	// +kubebuilder:validation:Required
	DHCP SubnetDHCP `json:"dhcp"`

	// VpcReference references the ArubaVpc that owns this subnet
	// +kubebuilder:validation:Required
	VpcReference ResourceReference `json:"vpcReference"`

	// ProjectReference references the Project that owns this block storage
	// +kubebuilder:validation:Required
	ProjectReference ResourceReference `json:"projectReference"`
}

// SubnetStatus defines the observed state of Subnet.
type SubnetStatus struct {
	ResourceStatus `json:",inline"`

	// ProjectID is the project ID where this subnet is created
	// +kubebuilder:validation:Optional
	ProjectID string `json:"projectID,omitempty"`

	// VpcID is the VPC ID where this subnet is created
	// +kubebuilder:validation:Optional
	VpcID string `json:"vpcID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=asn
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Resource ID",type="string",JSONPath=".status.resourceID"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Subnet is the Schema for the subnets API.
type Subnet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubnetSpec   `json:"spec,omitempty"`
	Status SubnetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SubnetList contains a list of Subnet.
type SubnetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Subnet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Subnet{}, &SubnetList{})
}
