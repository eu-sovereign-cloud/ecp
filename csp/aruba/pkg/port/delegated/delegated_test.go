package delegated

//go:generate mockgen -package delegated -destination=zz_mock_identifiable_test.go github.com/eu-sovereign-cloud/ecp/foundation/persistence/port IdentifiableResource
//go:generate mockgen -package delegated -destination=zz_mock_converter_test.go github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/converter Converter
