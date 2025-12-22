module github.com/eu-sovereign-cloud/ecp/foundation/plugin/aruba

go 1.24

require github.com/eu-sovereign-cloud/ecp/foundation/delegator v0.0.0

require github.com/Arubacloud/arubacloud-resource-operator v0.0.1-alpha4 // indirect

replace github.com/eu-sovereign-cloud/ecp/foundation/delegator => ../../delegator
