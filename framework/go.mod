module github.com/eu-sovereign-cloud/ecp/framework

go 1.26.4

tool sigs.k8s.io/controller-tools/cmd/controller-gen

require (
	github.com/gobwas/glob v0.2.3
	github.com/google/go-cmp v0.7.0
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.11.1
	golang.org/x/tools v0.44.0
	k8s.io/api v0.35.0
	k8s.io/apimachinery v0.35.0
	k8s.io/client-go v0.35.0
	sigs.k8s.io/controller-runtime v0.23.1
	sigs.k8s.io/controller-tools v0.20.0
)
