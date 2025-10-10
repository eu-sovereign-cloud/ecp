// +kubebuilder:object:generate=true
// +groupName=storage.v1.secapi.cloud
// +versionName=v1

package v1

import storage "github.com/eu-sovereign-cloud/ecp/apis/block-storage"

func init() {
	storage.SchemeBuilder.Register(&StorageSKU{}, &StorageSKUList{})
}
