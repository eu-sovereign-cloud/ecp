package aliasstatus

import "github.com/eu-sovereign-cloud/ecp/foundation/persistence/generated/types"

type AliasedStatus = types.Status

// +ecp:conditioned
type TypeWithAliasStatus struct {
	Status *AliasedStatus
}
