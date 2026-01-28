package handler

//go:generate mockgen -package=handler -destination=./zz_mock_workspace_repository.go github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/repository Repository,Writer,Watcher,Reader
//go:generate mockgen -package=handler -destination=./zz_mock_workspace_converter.go github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/converter Converter
