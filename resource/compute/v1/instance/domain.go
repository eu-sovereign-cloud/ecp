// Package instance defines the compute instance resource domain model and identity constants.
package instance

import (
	"time"

	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// Identity constants for the Instance resource.
const (
	Kind       = "Instance"
	Resource   = "instances"
	Group      = "compute.v1.secapi.cloud"
	Version    = "v1"
	ProviderID = "seca.compute/v1"
)

// PowerState represents the power state of an instance.
type PowerState string

// Power states for an instance.
const (
	PowerStateOn  PowerState = "on"
	PowerStateOff PowerState = "off"
)

// RestartPhase is a durable, monotonically-advancing phase of a restart cycle. A restart
// progresses power-off -> power-on -> (cleared). The phase is persisted on the CR so the
// delegator can resume mid-cycle after a crash, rather than inferring progress from the
// current power state.
type RestartPhase string

// Restart phases.
const (
	RestartPhasePowerOff RestartPhase = "power-off"
	RestartPhasePowerOn  RestartPhase = "power-on"
)

// Instance represents the domain model for a compute instance.
type Instance struct {
	domain.RegionalMetadata
	Spec   InstanceSpec
	Status *InstanceStatus

	// DesiredPowerState is the power state requested via start/stop. It is internal
	// control state carried on the CR (not exposed through the API) and is consumed
	// by the delegator to drive Status.PowerState. Empty means no explicit request.
	DesiredPowerState PowerState
	// RestartID identifies an in-flight restart. The gateway sets a fresh value per
	// restart request; the delegator clears it (together with RestartPhase) once the
	// cycle completes. It also lets the delegator detect a superseding restart so it
	// does not clobber a newer request during cleanup. Internal, not exposed via the API.
	RestartID string
	// RestartPhase is the durable phase of the in-flight restart (empty when none).
	// Internal control state carried on the CR, not exposed through the API.
	RestartPhase RestartPhase
}

// VolumeReference represents a connection between a block storage and a user of the block storage.
type VolumeReference struct {
	DeviceRef domain.Reference
	// Type is the connection type. Empty means unset (provider default).
	Type string
}

// InstanceSpec defines the specification for an Instance.
type InstanceSpec struct {
	// AdditionalNicRefs are additional NICs attached to this instance.
	AdditionalNicRefs []domain.Reference
	// AntiAffinityGroup is the anti-affinity group to which this instance belongs. Empty means unset.
	AntiAffinityGroup string
	// BootVolume references the block storage used for the boot volume.
	BootVolume VolumeReference
	// DataVolumes references additional block storage volumes.
	DataVolumes []VolumeReference
	// PrimaryNicRef is optional. Nil means unset.
	PrimaryNicRef *domain.Reference
	// SecurityGroupRef is optional. Nil means unset.
	SecurityGroupRef *domain.Reference
	// SkuRef references the SKU of the instance. It is immutable after creation.
	SkuRef domain.Reference
	// SshKeys are provider-specific references to SSH keys used in cloud-init vendorData.
	SshKeys []string
	// UserData is cloud-init user data for instance initialization. Empty means unset.
	UserData string
	// Zone is the zone in which the instance is deployed. It is immutable after creation.
	Zone string
}

// InstanceStatus defines the status for an Instance.
type InstanceStatus struct {
	domain.Status
	// PowerState is the current power state of the instance.
	PowerState PowerState
	// PowerStateSince is the time the power state last changed. Nil if the instance was never started.
	PowerStateSince *time.Time
}
