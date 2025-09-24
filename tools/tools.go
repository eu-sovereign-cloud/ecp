//go:build tools
// +build tools

package main

import (
	_ "github.com/crossplane/crossplane-tools/cmd/angryjet"
	_ "github.com/mproffitt/crossbuilder/cmd/xrd-gen"
	_ "github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
