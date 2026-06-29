package kubernetes

//go:generate go run github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/cmd/model-gen --schema-file=../../../../../../modules/go-sdk/pkg/spec/schema/image.go --output-file=zz_generated_schema.go --package-name=kubernetes --root-types=ImageSpec,ImageStatus --shared-types-source=../../../../../../modules/go-sdk/pkg/spec/schema/resource.go
//go:generate go run go.uber.org/mock/mockgen -package kubernetes_test -destination ./zz_mock_repo_test.go github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence Repo
//go:generate go run go.uber.org/mock/mockgen -package kubernetes_test -destination ./zz_mock_plugin_test.go github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes ImagePlugin
//go:generate go run go.uber.org/mock/mockgen -package kubernetes_test -destination ./zz_mock_deps_test.go github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes DependencyResolver
