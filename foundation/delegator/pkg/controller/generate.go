package controller

//go:generate mockgen -package controller -destination=zz_mock_blockstorage_repo_test.go github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port Repo
//go:generate mockgen -package controller -destination=zz_mock_blockstorage_plugin_test.go github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin BlockStorage
//go:generate mockgen -package controller -destination=zz_mock_workspace_plugin_test.go github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin Workspace
//go:generate mockgen -package controller -destination=zz_mock_plugin_handler_test.go github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/port PluginHandler
//go:generate mockgen -package controller -destination=zz_mock_client_test.go sigs.k8s.io/controller-runtime/pkg/client Client
//go:generate mockgen -package controller -destination=zz_mock_status_writer_test.go sigs.k8s.io/controller-runtime/pkg/client StatusWriter
