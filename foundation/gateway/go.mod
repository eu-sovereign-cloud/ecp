module github.com/eu-sovereign-cloud/ecp/foundation/gateway

go 1.24

toolchain go1.24

require (
	github.com/eu-sovereign-cloud/ecp/foundation/api v0.0.1
	github.com/eu-sovereign-cloud/go-sdk v0.2.0
	github.com/gobwas/glob v0.2.3
	github.com/spf13/cobra v1.10.1
	github.com/stretchr/testify v1.11.1
	k8s.io/api v0.34.3
	k8s.io/apimachinery v0.34.3
	k8s.io/client-go v0.34.3
	k8s.io/utils v0.0.0-20251002143259-bc988d571ff4
	sigs.k8s.io/controller-runtime v0.22.4
)
