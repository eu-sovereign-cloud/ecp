//go:build crdgen

// Package network contains go:generate directives for CRD and DeepCopy generation.
// Build with -tags=crdgen to activate.
package network

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../../.github/boilerplate.go.txt paths=./skus/v1/...
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./skus/v1/... output:crd:artifacts:config=../../generated/crds/network
