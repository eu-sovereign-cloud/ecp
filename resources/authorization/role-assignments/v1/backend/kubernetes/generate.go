package kubernetes

//go:generate go run github.com/eu-sovereign-cloud/ecp/framework/persistence/cmd/model-gen --schema-file=../../../../../../modules/go-sdk/pkg/spec/schema/rbac.go --output-file=zz_generated_schema.go --package-name=kubernetes --root-types=RoleAssignmentSpec,RoleAssignmentStatus --shared-types-source=../../../../../../modules/go-sdk/pkg/spec/schema/resource.go
