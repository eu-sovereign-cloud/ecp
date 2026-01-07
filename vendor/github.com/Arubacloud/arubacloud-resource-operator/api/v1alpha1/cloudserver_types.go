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

// CloudServerSpec defines the desired state of CloudServer.
type CloudServerSpec struct {
	// Tenant is the owning account/tenant of this cloud server
	Tenant string `json:"tenant,omitempty"`

	// Tags are labels associated with the cloud server
	// +kubebuilder:validation:Optional
	Tags []string `json:"tags,omitempty"`

	// Location specifies the location for the cloud server
	// +kubebuilder:validation:Required
	Location Location `json:"location"`

	// DataCenter specifies the data center
	// +kubebuilder:validation:Required
	DataCenter string `json:"dataCenter"`

	// VpcReference references the VPC where the cloud server will be created
	// +kubebuilder:validation:Required
	VpcReference ResourceReference `json:"vpcReference"`

	// VpcPreset indicates whether to use VPC preset
	// +kubebuilder:validation:Optional
	VpcPreset bool `json:"vpcPreset,omitempty"`

	// FlavorId specifies the flavor/size of the cloud server
	// +kubebuilder:validation:Required
	FlavorName string `json:"flavorName"`

	// ElasticIpReference references an existing elastic IP (optional)
	// +kubebuilder:validation:Optional
	ElasticIpReference *ResourceReference `json:"elasticIpReference,omitempty"`

	// KeyPairReference references a key pair for SSH access (optional)
	// +kubebuilder:validation:Required
	KeyPairReference ResourceReference `json:"keyPairReference"`

	// SubnetReferences references the subnets where the cloud server will be attached
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	SubnetReferences []ResourceReference `json:"subnetReferences"`

	// SecurityGroupReferences references the security groups for the cloud server
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	SecurityGroupReferences []ResourceReference `json:"securityGroupReferences"`

	// BootVolumeReference references the boot volume for the cloud server
	// +kubebuilder:validation:Required
	BootVolumeReference ResourceReference `json:"bootVolumeReference"`

	// DataVolumeReferences references additional data volumes to attach to the cloud server (optional)
	// +kubebuilder:validation:Optional
	DataVolumeReferences []ResourceReference `json:"dataVolumeReferences,omitempty"`

	// ProjectReference references the Project that owns this cloud server
	// +kubebuilder:validation:Required
	ProjectReference ResourceReference `json:"projectReference"`
}

// CloudServerStatus defines the observed state of CloudServer.
type CloudServerStatus struct {
	ResourceStatus `json:",inline"`

	// ProjectID is the project ID where this cloud server is created
	// +kubebuilder:validation:Optional
	ProjectID string `json:"projectID,omitempty"`

	// VpcID is the VPC ID where this cloud server is created
	// +kubebuilder:validation:Optional
	VpcID string `json:"vpcID,omitempty"`

	// BootVolumeID is the boot volume ID where this cloud server is created
	// +kubebuilder:validation:Optional
	BootVolumeID string `json:"bootVolumeID,omitempty"`

	// ElasticIpID is the elastic IP ID if one is assigned
	// +kubebuilder:validation:Optional
	ElasticIpID string `json:"elasticIpID,omitempty"`

	// KeyPairID is the key pair ID if one is specified
	// +kubebuilder:validation:Optional
	KeyPairID string `json:"keyPairID,omitempty"`

	// SubnetIDs are the subnet IDs where this cloud server is attached
	// +kubebuilder:validation:Optional
	SubnetIDs []string `json:"subnetIDs,omitempty"`

	// SecurityGroupIDs are the security group IDs for this cloud server
	// +kubebuilder:validation:Optional
	SecurityGroupIDs []string `json:"securityGroupIDs,omitempty"`

	// DataVolumeIDs are the data volume IDs attached to this cloud server
	// +kubebuilder:validation:Optional
	DataVolumeIDs []string `json:"dataVolumeIDs,omitempty"`

	// VolumeIDs are the volume IDs attached to this cloud server
	// +kubebuilder:validation:Optional
	VolumeIDs []string `json:"volumeIDs,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=cs
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Resource ID",type="string",JSONPath=".status.resourceID"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// CloudServer is the Schema for the cloudservers API.
type CloudServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CloudServerSpec   `json:"spec,omitempty"`
	Status CloudServerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// CloudServerList contains a list of CloudServer.
type CloudServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CloudServer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CloudServer{}, &CloudServerList{})
}
