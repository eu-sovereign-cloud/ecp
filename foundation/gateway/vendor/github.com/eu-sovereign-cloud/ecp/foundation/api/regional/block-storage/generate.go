//go:build crdgen

package network

<<<<<<<< HEAD:foundation/gateway/vendor/github.com/eu-sovereign-cloud/ecp/foundation/api/regional/block-storage/generate.go
//go:generate go run -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../../.github/boilerplate.go.txt paths=./skus/v1/...
//go:generate go run -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./skus/v1/... output:crd:artifacts:config=../../generated/crds/block-storage
========
//go:generate go run -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../.github/boilerplate.go.txt paths=./skus/v1/...
//go:generate go run -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./skus/v1/... output:crd:artifacts:config=../generated/crds/network
>>>>>>>> main:foundation/gateway/vendor/github.com/eu-sovereign-cloud/ecp/foundation/api/network/generate.go
