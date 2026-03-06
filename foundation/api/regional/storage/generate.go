//go:build crdgen

// Package storage contains go:generate directives for CRD and DeepCopy generation.
// Build with -tags=crdgen to activate.
package storage

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../../.github/boilerplate.go.txt paths=./skus/v1/...
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./skus/v1/... output:crd:artifacts:config=../../generated/crds/storage

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../../.github/boilerplate.go.txt paths=./block_storages/v1/...
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./block_storages/v1/... output:crd:artifacts:config=../../generated/crds/storage
