package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/common"
	"github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/api/regional/storage"
	genv1 "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/generated/types"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=images,scope=Namespaced,shortName=image
// +k8s:openapi-gen=true
// +ecp:conditioned

// Image is the API for managing compute images.
type Image struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec       genv1.ImageSpec    `json:"spec,omitempty"`
	CommonData common.CommonData  `json:"commonData,omitempty"`
	Status     *genv1.ImageStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Image `json:"items"`
}

func init() {
	storage.SchemeBuilder.Register(&Image{}, &ImageList{})
}
