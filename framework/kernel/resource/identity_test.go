package resource_test

import (
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
)

// These assertions live in an external test package (resource_test) rather than
// beside the type because the persistence port imports this package — a
// same-package assertion would create an import cycle.
var (
	_ persistence.IdentifiableResource = resource.Identity{}
	_ persistence.IdentifiableResource = (*resource.Identity)(nil)
)
