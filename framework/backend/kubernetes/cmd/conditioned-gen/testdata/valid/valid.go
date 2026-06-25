package valid

import v1 "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/schema/v1"

// +ecp:conditioned
type TypeOne struct {
	Status *v1.Status
}

type (
	// +ecp:conditioned
	TypeTwo struct {
		Status *v1.Status
	}
)
