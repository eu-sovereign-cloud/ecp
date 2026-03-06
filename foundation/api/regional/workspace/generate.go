//go:build crdgen

// Package workspace contains go:generate directives for CRD and DeepCopy generation.
// Build with -tags=crdgen to activate.
package workspace

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../../.github/boilerplate.go.txt paths=./v1/...
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./v1/... output:crd:artifacts:config=../../generated/crds/workspace
