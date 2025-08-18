//go:build generate

// Generate crossplane-runtime methodsets (resource.Claim, etc)
//go:generate go run -tags generate github.com/crossplane/crossplane-tools/cmd/angryjet generate-methodsets --header-file=.github/boilerplate.go.txt ./...

package main

import _ "github.com/crossplane/crossplane-tools/cmd/angryjet"
