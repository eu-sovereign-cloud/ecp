package aliasstatus

import v1 "github.com/eu-sovereign-cloud/ecp/framework/persistence/kubernetes/schema/v1"

type AliasedStatus = v1.Status

// +ecp:conditioned
type TypeWithAliasStatus struct {
	Status *AliasedStatus
}
