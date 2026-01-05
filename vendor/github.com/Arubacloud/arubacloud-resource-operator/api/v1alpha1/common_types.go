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

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// Common phases for all resources
type ResourcePhase string

const (
	// ResourcePhaseCreating indicates the resource is being created
	ResourcePhaseCreating ResourcePhase = "Creating"
	// ResourcePhaseProvisioning indicates the resource is being provisioned remotely
	ResourcePhaseProvisioning ResourcePhase = "Provisioning"
	// ResourcePhaseCreated indicates the resource has been created successfully
	ResourcePhaseCreated ResourcePhase = "Created"
	// ResourcePhaseUpdating indicates the resource is being updated
	ResourcePhaseUpdating ResourcePhase = "Updating"
	// ResourcePhaseDeleting indicates the resource is being deleted
	ResourcePhaseDeleting ResourcePhase = "Deleting"
	// ResourcePhaseDeleted indicates the resource has been deleted
	ResourcePhaseDeleted ResourcePhase = "Deleted"
	// ResourcePhaseFailed indicates the resource has failed
	ResourcePhaseFailed ResourcePhase = "Failed"
)

// Condition types for resources
const (
	// ConditionTypeSynchronized indicates whether the resource is synchronized with the remote system
	ConditionTypeSynchronized = "Synchronized"
)

// Location specifies the location for resources
type Location struct {
	// Value is the location identifier (e.g., "ITBG-Bergamo")
	// +kubebuilder:validation:Required
	Value string `json:"value"`
}

// ResourceReference represents a reference to another resource
type ResourceReference struct {
	// Name is the name of the referenced resource
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Namespace is the namespace of the referenced resource
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace,omitempty"`
}

// Common status for all resources
type ResourceStatus struct {
	// Phase represents the current phase of the resource
	// +kubebuilder:validation:Optional
	Phase ResourcePhase `json:"phase,omitempty"`

	// Message provides human-readable information about the current state
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitempty"`

	// ResourceID is the unique identifier of the resource in the remote system
	// +kubebuilder:validation:Optional
	ResourceID string `json:"resourceID,omitempty"`

	// ObservedGeneration is the most recent generation observed by the controller
	// +kubebuilder:validation:Optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// PhaseStartTime tracks when the current phase started
	// +kubebuilder:validation:Optional
	PhaseStartTime *metav1.Time `json:"phaseStartTime,omitempty"`

	// Conditions represent the latest available observations of the Resource state
	// +listType=map
	// +listMapKey=type
	// +patchStrategy=merge
	// +patchMergeKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// Object is the common Schema for the API.
type Object struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Status ResourceStatus `json:"status,omitempty"`
}
