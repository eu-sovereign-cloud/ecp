//go:build xrdgen

package blockstorage

//go:generate go run -mod=mod sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../.github/boilerplate.go.txt paths=.
