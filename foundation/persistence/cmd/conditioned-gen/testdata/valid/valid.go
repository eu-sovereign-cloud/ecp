package valid

import "github.com/eu-sovereign-cloud/ecp/foundation/persistence/generated/types"

// +ecp:conditioned
type TypeOne struct {
	Status *types.Status
}

type (
	// +ecp:conditioned
	TypeTwo struct {
		Status *types.Status
	}
)
