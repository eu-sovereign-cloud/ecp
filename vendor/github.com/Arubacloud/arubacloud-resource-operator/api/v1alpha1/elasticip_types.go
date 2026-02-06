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

// BillingPlan represents the billing configuration
type BillingPlan struct {
	// BillingPeriod defines the billing period (Hour, Month, etc.)
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=Hour;Month
	BillingPeriod string `json:"billingPeriod"`
}

// ElasticIpSpec defines the desired state of ElasticIp.
type ElasticIpSpec struct {
	// Tenant is the owning account/tenant of this elastic IP
	Tenant string `json:"tenant,omitempty"`

	// Tags are labels associated with the elastic IP
	// +kubebuilder:validation:Optional
	Tags []string `json:"tags,omitempty"`

	// Location specifies the location for the elastic IP
	// +kubebuilder:validation:Required
	Location Location `json:"location"`

	// BillingPlan specifies the billing configuration
	// +kubebuilder:validation:Required
	BillingPlan BillingPlan `json:"billingPlan"`

	// ProjectReference references the Project that owns this elastic IP
	// +kubebuilder:validation:Required
	ProjectReference ResourceReference `json:"projectReference"`
}

// ElasticIpStatus defines the observed state of ElasticIp.
type ElasticIpStatus struct {
	ResourceStatus `json:",inline"`

	// ProjectID is the project ID where this elastic IP is created
	// +kubebuilder:validation:Optional
	ProjectID string `json:"projectID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=eip
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Resource ID",type="string",JSONPath=".status.resourceID"
// +kubebuilder:printcolumn:name="Message",type="string",JSONPath=".status.message"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ElasticIp is the Schema for the elasticips API.
type ElasticIp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticIpSpec   `json:"spec,omitempty"`
	Status ElasticIpStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ElasticIpList contains a list of ElasticIp.
type ElasticIpList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticIp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElasticIp{}, &ElasticIpList{})
}
