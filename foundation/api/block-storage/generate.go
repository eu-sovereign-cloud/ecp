//go:build crdgen

package blockstorage

//go:generate go run -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../.github/boilerplate.go.txt paths=./skus/v1/...
//go:generate go run -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./skus/v1/... output:crd:artifacts:config=../generated/crds/block-storage
