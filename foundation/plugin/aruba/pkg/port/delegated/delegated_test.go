package delegated

//go:generate mockgen -package delegated -destination=zz_mock_identifiable_test.go github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port IdentifiableResource
//go:generate mockgen -package delegated -destination=zz_mock_converter_test.go github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba/pkg/port/converter Converter
