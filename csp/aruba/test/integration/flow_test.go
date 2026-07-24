//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
)

// TestArubaFlow drives every SECA resource the aruba plugin reconciles, in dependency order, and
// asserts each reaches Active AND the matching arubacloud.com CR the plugin materialises does too.
// It is one ordered test, not per-resource parallel like the dummy suite, because aruba's resources
// form a dependency graph (a subnet needs its VPC active, an instance needs its whole graph) and the
// backend provisions them for real. The instance step asserts materialisation rather than a running
// VM (see its comment).
func TestArubaFlow(t *testing.T) {
	t.Cleanup(deleteFlowResources) // don't leak real Aruba resources, even on failure

	// 1. workspace -> aruba Project
	mustActive(t, wsRepo, newWorkspace(workspace), activeTimeout, func() (commondomain.ResourceState, bool) {
		o := newWorkspace(workspace)
		if wsRepo.Load(ctx, &o) != nil || o.Status == nil {
			return "", false
		}
		return o.Status.State, true
	})
	requireArubaActive(t, "projects", workspace, tenantNS)

	// 2. block-storage -> aruba BlockStorage
	mustActive(t, bsRepo, newBlockStorage(bootName), activeTimeout, func() (commondomain.ResourceState, bool) {
		o := newBlockStorage(bootName)
		if bsRepo.Load(ctx, &o) != nil || o.Status == nil {
			return "", false
		}
		return o.Status.State, true
	})
	requireArubaActive(t, "blockstorages", bootName, wsNS)

	// 3. GATING: a network with no internet-gateway must not even get a VPC created - the aruba
	// NetworkHandler requires an InternetGateway to exist first. Create the network, then assert no
	// arubacloud VPC appears while the gate holds.
	_, err := netRepo.Create(ctx, newNetwork(network))
	require.NoError(t, err)
	requireStaysGated(t, "vpcs", network, wsNS)

	// 4. internet-gateway -> no-op (and the precondition that releases the gate)
	mustActive(t, igwRepo, newInternetGateway(igwName), noopTimeout, func() (commondomain.ResourceState, bool) {
		o := newInternetGateway(igwName)
		if igwRepo.Load(ctx, &o) != nil || o.Status == nil {
			return "", false
		}
		return o.Status.State, true
	})
	requireNoArubaCR(t, "internetgateways", igwName, wsNS)

	// 5. network -> aruba VPC: with the gate released, the VPC is now created and provisions.
	mustActive(t, netRepo, newNetwork(network), activeTimeout, func() (commondomain.ResourceState, bool) {
		o := newNetwork(network)
		if netRepo.Load(ctx, &o) != nil || o.Status == nil {
			return "", false
		}
		return o.Status.State, true
	})
	requireArubaActive(t, "vpcs", network, wsNS)

	// 6. route-table -> no-op
	mustActive(t, rtRepo, newRouteTable(rtName), noopTimeout, func() (commondomain.ResourceState, bool) {
		o := newRouteTable(rtName)
		if rtRepo.Load(ctx, &o) != nil || o.Status == nil {
			return "", false
		}
		return o.Status.State, true
	})

	// 7. subnet -> aruba Subnet
	mustActive(t, subRepo, newSubnet(subnetName), activeTimeout, func() (commondomain.ResourceState, bool) {
		o := newSubnet(subnetName)
		if subRepo.Load(ctx, &o) != nil || o.Status == nil {
			return "", false
		}
		return o.Status.State, true
	})
	requireArubaActive(t, "subnets", subnetName, netNS)

	// 8. public-ip -> aruba ElasticIP
	mustActive(t, pipRepo, newPublicIp(pipName), activeTimeout, func() (commondomain.ResourceState, bool) {
		o := newPublicIp(pipName)
		if pipRepo.Load(ctx, &o) != nil || o.Status == nil {
			return "", false
		}
		return o.Status.State, true
	})
	requireArubaActive(t, "elasticips", pipName, wsNS)

	// 9. security-group + 10. rule + 11. nic -> no-ops (the real Aruba SG is materialised at attach)
	mustActive(t, sgRepo, newSecurityGroup(sgName), noopTimeout, func() (commondomain.ResourceState, bool) {
		o := newSecurityGroup(sgName)
		if sgRepo.Load(ctx, &o) != nil || o.Status == nil {
			return "", false
		}
		return o.Status.State, true
	})
	mustActive(t, sgrRepo, newSecurityGroupRule(sgrName), noopTimeout, func() (commondomain.ResourceState, bool) {
		o := newSecurityGroupRule(sgrName)
		if sgrRepo.Load(ctx, &o) != nil || o.Status == nil {
			return "", false
		}
		return o.Status.State, true
	})
	mustActive(t, nicRepo, newNic(nicName), noopTimeout, func() (commondomain.ResourceState, bool) {
		o := newNic(nicName)
		if nicRepo.Load(ctx, &o) != nil || o.Status == nil {
			return "", false
		}
		return o.Status.State, true
	})

	// 12. instance: assert the plugin MATERIALISES the CloudServer graph - the per-VPC SecurityGroup
	// and the KeyPair provision, and the CloudServer CR is created with them referenced. The VM
	// itself is not asserted Active: Aruba's CloudServer flavor is the SECA compute-SKU name verbatim
	// (compute-sku-1 here), which is not a real Aruba flavor, so provisioning is a catalog concern
	// outside the plugin's contract.
	_, err = instRepo.Create(ctx, newInstance(instName, envOr("ARUBA_SSH_KEY", defaultSSHKey)))
	require.NoError(t, err)
	// The materialised security group + its rule (named <sg>-<network>) provision for real.
	matSG := sgName + "-" + network
	requireEventuallyActive(t, "securitygroups", matSG, wsNS)
	requireEventuallyActive(t, "keypairs", instName+keyPairSuffix, wsNS)
	// The CloudServer CR itself is created (materialisation), regardless of provisioning outcome.
	requireEventuallyPresent(t, "cloudservers", instName, wsNS)
}

// keyPairSuffix mirrors converter.KeyPairSuffix ("-key") without importing the plugin internals.
const keyPairSuffix = "-key"

// requireStaysGated asserts the arubacloud CR does NOT appear for a short window - the gate holds.
func requireStaysGated(t *testing.T, resource, name, ns string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		require.Equalf(t, "NOTFOUND", arubaPhase(resource, name, ns),
			"%s should not be created while the internet-gateway gate holds", resource)
		time.Sleep(pollInterval)
	}
}

func requireEventuallyActive(t *testing.T, resource, name, ns string) {
	t.Helper()
	require.NoErrorf(t, wait.PollUntilContextTimeout(ctx, pollInterval, activeTimeout, true,
		func(context.Context) (bool, error) { return arubaPhase(resource, name, ns) == "Active", nil }),
		"arubacloud.com %s/%s should reach Active", resource, name)
}

func requireEventuallyPresent(t *testing.T, resource, name, ns string) {
	t.Helper()
	require.NoErrorf(t, wait.PollUntilContextTimeout(ctx, pollInterval, activeTimeout, true,
		func(context.Context) (bool, error) { return arubaPhase(resource, name, ns) != "NOTFOUND", nil }),
		"arubacloud.com %s/%s should be created", resource, name)
}
