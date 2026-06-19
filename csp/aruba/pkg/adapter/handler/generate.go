package handler

//go:generate mockgen -package=handler -destination=./zz_mock_workspace_repository.go github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/repository Repository,Writer,Watcher,Reader
//go:generate mockgen -package=handler -destination=./zz_mock_workspace_converter.go github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/converter Converter
//go:generate mockgen -package=handler -destination=./zz_mock_reader_repo.go github.com/eu-sovereign-cloud/ecp/foundation/persistence/port ReaderRepo
