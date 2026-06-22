package kubernetes

//go:generate go run github.com/eu-sovereign-cloud/ecp/framework/persistence/cmd/model-gen --schema-file=../../../../../../../modules/go-sdk/pkg/spec/schema/public-ip.go --output-file=zz_generated_types.go --package-name=kubernetes --root-types=PublicIpSpec,PublicIpStatus --shared-types-source=../../../../../../../modules/go-sdk/pkg/spec/schema/resource.go
