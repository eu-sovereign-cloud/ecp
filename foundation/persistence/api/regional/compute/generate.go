//go:build crdgen

package compute

//go:generate go run ../../../cmd/conditioned-gen --header-file=../../../../../.github/boilerplate.go.txt --paths=./instances/v1/...
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../../../.github/boilerplate.go.txt paths=./instances/v1/...
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./instances/v1/... output:crd:artifacts:config=../../../generated/crds/compute

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../../../.github/boilerplate.go.txt paths=./skus/v1/...
//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./skus/v1/... output:crd:artifacts:config=../../../generated/crds/compute
