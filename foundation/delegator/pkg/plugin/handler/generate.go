package handler

//go:generate mockgen -package handler -destination=zz_mock_repo_test.go github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port Repo
//go:generate mockgen -package handler -destination=zz_mock_blockstorage_plugin_test.go github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin BlockStorage
//go:generate mockgen -package handler -destination=zz_mock_workspace_plugin_test.go github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin Workspace
