//go:build tools
// +build tools

package main

import (
	_ "git"

	_ "github.com/crossplane/crossplane-tools/cmd/angryjet"
	_ "github.com/mproffitt/crossbuilder/cmd/xrd-gen"
	_ "sigs.k8s.io/controller-tools/cmd/controller-gen"
)
