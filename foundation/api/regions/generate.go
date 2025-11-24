//go:build crdgen

package regions

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../.github/boilerplate.go.txt paths=./v1/...
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./v1/... output:crd:artifacts:config=../generated/crds/regions
