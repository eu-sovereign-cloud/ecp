// +kubebuilder:object:generate=true
// +groupName=storage.v1.secapi.cloud
// +versionName=v1

package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"

	schemav1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"
	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	imagedomain "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
)

const (
	ImageResource = imagedomain.Resource
	ImageKind     = imagedomain.Kind
)

var (
	GroupVersion  = schema.GroupVersion{Group: imagedomain.Group, Version: imagedomain.Version}
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
	AddToScheme   = SchemeBuilder.AddToScheme

	ImageGVR = domain.GVR(imagedomain.Group, imagedomain.Version, imagedomain.Resource)
	ImageGVK = domain.GVK(imagedomain.Group, imagedomain.Version, imagedomain.Kind)
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

	Spec       ImageSpec           `json:"spec,omitempty"`
	CommonData schemav1.CommonData `json:"commonData,omitempty"`
	Status     *ImageStatus        `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type ImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Image `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Image{}, &ImageList{})
}
