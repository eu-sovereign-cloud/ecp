//go:build xrdgen

package blockstorage

//go:generate go run -mod=mod github.com/mproffitt/crossbuilder/cmd/xrd-gen object:headerFile=../../../.github/boilerplate.go.txt paths=./storages/v1/...
//go:generate go run -mod=mod github.com/mproffitt/crossbuilder/cmd/xrd-gen xrd paths=./storages/v1/... output:xrd:artifacts:config=../../generated/xrds/block-storages
