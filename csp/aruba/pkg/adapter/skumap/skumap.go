// Package skumap maps a SECA SKU's abstract capacity to the concrete Aruba catalog value the
// arubacloud-resource-operator expects. SECA SKUs describe capacity (vCPU/RAM, IOPS); Aruba names a
// fixed catalog of flavors and storage tiers. The catalog is small and stable, so it is embedded
// here rather than fetched: the delegator holds no Aruba credentials (only the operator does), so it
// cannot query Aruba's live catalog.
//
// Source of truth for the values: github.com/Arubacloud/sdk-go pkg/types — CloudServerFlavor (the
// CSO<cpu>A<ram> enum) and BlockStorageType (Standard/Performance).
package skumap

import "fmt"

// arubaFlavor is one row of the embedded Aruba CloudServer flavor catalog: a name and the (vCPU, RAM
// in GB) it provides. RAM is in GB to match the SECA InstanceSKU (Aruba's API reports RAM in MB, but
// the flavor name encodes GB - CSO4A8 is 4 vCPU / 8 GB).
type arubaFlavor struct {
	name  string
	vcpu  int
	ramGB int
}

// computeFlavors is the Aruba CloudServer flavor enum (sdk-go CloudServerFlavor). Keep in step with
// the SDK when Aruba adds flavors.
var computeFlavors = []arubaFlavor{
	{"CSO1A2", 1, 2},
	{"CSO1A4", 1, 4},
	{"CSO2A4", 2, 4},
	{"CSO2A8", 2, 8},
	{"CSO4A8", 4, 8},
	{"CSO4A16", 4, 16},
	{"CSO8A16", 8, 16},
	{"CSO8A32", 8, 32},
	{"CSO12A24", 12, 24},
	{"CSO16A32", 16, 32},
	{"CSO16A64", 16, 64},
	{"CSO24A48", 24, 48},
	{"CSO32A64", 32, 64},
}

// ComputeFlavor returns the Aruba CloudServer flavor whose capacity exactly matches a SECA
// InstanceSKU's vCPU and RAM (GB). It errors when no flavor matches, so an unsatisfiable SKU fails
// loudly instead of sending a bad FlavorName to Aruba (a 400 with no useful detail).
func ComputeFlavor(vcpu, ramGB int) (string, error) {
	for _, f := range computeFlavors {
		if f.vcpu == vcpu && f.ramGB == ramGB {
			return f.name, nil
		}
	}
	return "", fmt.Errorf("no Aruba CloudServer flavor provides %d vCPU / %d GB RAM", vcpu, ramGB)
}

// Aruba BlockStorage tiers (sdk-go BlockStorageType).
const (
	storageStandard    = "Standard"
	storagePerformance = "Performance"

	// performanceIOPSFloor is the SECA IOPS at or above which a volume maps to Aruba's Performance
	// tier. Aruba exposes only Standard/Performance and does not publish the tier's IOPS boundary,
	// so this is a tunable heuristic keyed on the objective capacity (IOPS), not the SECA Type
	// string (which describes durability/locality, e.g. "local-durable", not a perf tier).
	// ponytail: single threshold; refine to a table if Aruba documents per-tier IOPS.
	performanceIOPSFloor = 10000
)

// StorageType maps a SECA StorageSKU's IOPS to an Aruba block-storage tier. It never errors: both
// tiers are always valid, and Aruba defaults the field when empty, so a coarse choice is safe.
func StorageType(iops int64) string {
	if iops >= performanceIOPSFloor {
		return storagePerformance
	}
	return storageStandard
}
