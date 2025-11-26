//go:build crdgen
package common

//go:generate go run sigs.k8s.io/controller-tools/cmd/controller-gen object:headerFile=../../../../.github/boilerplate.go.txt paths=./...

