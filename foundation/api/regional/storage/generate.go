//go:build crdgen

package storage

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../../.github/boilerplate.go.txt paths=./skus/v1/...
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./skus/v1/... output:crd:artifacts:config=../../generated/crds/storage

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../../.github/boilerplate.go.txt paths=./block-storages/v1/...
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./block-storages/v1/... output:crd:artifacts:config=../../generated/crds/storage
