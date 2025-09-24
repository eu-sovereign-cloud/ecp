// +kubebuilder:object:generate=true
// +groupName=v1.secapi.cloud
// +versionName=v1

package v1

import "github.com/eu-sovereign-cloud/ecp/apis/storage"

func init() {
	storage.SchemeBuilder.Register(&BlockStorage{}, &BlockStorageList{})
}