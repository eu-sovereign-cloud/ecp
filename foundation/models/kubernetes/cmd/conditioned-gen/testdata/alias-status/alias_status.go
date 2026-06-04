package aliasstatus

import "github.com/eu-sovereign-cloud/ecp/foundation/models/kubernetes/generated/types"

type AliasedStatus = types.Status

// +ecp:conditioned
type TypeWithAliasStatus struct {
	Status *AliasedStatus
}
