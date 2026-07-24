//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	kres "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instdom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	igwdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	pipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
	rtdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
	sgdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	sgrdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	subdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// ---- metadata + reference builders ----------------------------------------

func rMeta(name string) commondomain.RegionalMetadata {
	return commondomain.RegionalMetadata{
		CommonMetadata: commondomain.CommonMetadata{Name: name},
		Scope:          kres.Scope{Tenant: tenant, Workspace: workspace},
		Region:         region,
	}
}

// tenant-scoped metadata (workspace itself is identified by tenant + name).
func tMeta(name string) commondomain.RegionalMetadata {
	m := rMeta(name)
	m.Scope = kres.Scope{Tenant: tenant}
	return m
}

func rnMeta(name string) commondomain.RegionalNetworkMetadata {
	return commondomain.RegionalNetworkMetadata{RegionalMetadata: rMeta(name), Network: network}
}

func ref(resource string) commondomain.Reference { return commondomain.Reference{Resource: resource} }
func ptr[T any](v T) *T                          { return &v }

// ---- resource builders (aruba-valid: names >= 4 chars, required CRD fields set) ----

func newWorkspace(name string) *wsdom.Workspace {
	return &wsdom.Workspace{RegionalMetadata: tMeta(name)}
}

func newBlockStorage(name string) *bsdom.BlockStorage {
	return &bsdom.BlockStorage{RegionalMetadata: rMeta(name),
		Spec: bsdom.BlockStorageSpec{SizeGB: 10, SkuRef: ref("sku-1")}}
}

func newInternetGateway(name string) *igwdom.InternetGateway {
	return &igwdom.InternetGateway{RegionalMetadata: rMeta(name)}
}

func newRouteTable(name string) *rtdom.RouteTable {
	return &rtdom.RouteTable{RegionalNetworkMetadata: rnMeta(name),
		Spec: rtdom.RouteTableSpec{Routes: []rtdom.RouteSpec{
			{DestinationCidrBlock: "10.30.0.0/16", TargetRef: ref("internet-gateways/" + igwName)}}}}
}

func newNetwork(name string) *netdom.Network {
	return &netdom.Network{RegionalMetadata: rMeta(name),
		Spec: netdom.NetworkSpec{CIDR: netdom.CIDR{IPv4: "10.30.0.0/16"}, SkuRef: ref("network-sku-1")}}
}

func newSubnet(name string) *subdom.Subnet {
	return &subdom.Subnet{RegionalNetworkMetadata: rnMeta(name),
		Spec: subdom.SubnetSpec{Cidr: subdom.CIDR{IPv4: "10.30.1.0/24"},
			RouteTableRef: ref("route-tables/" + rtName), Zone: "ITBG-1"}}
}

func newPublicIp(name string) *pipdom.PublicIp {
	return &pipdom.PublicIp{RegionalMetadata: rMeta(name),
		Spec: pipdom.PublicIpSpec{Version: commondomain.IPVersionIPv4}}
}

func newSecurityGroup(name string) *sgdom.SecurityGroup {
	return &sgdom.SecurityGroup{RegionalMetadata: rMeta(name),
		Spec: sgdom.SecurityGroupSpec{Rules: []sgdom.SecurityGroupRuleSpec{
			{Direction: "ingress", Protocol: "tcp", Ports: &sgdom.Ports{From: 22, To: 22}}}}}
}

func newSecurityGroupRule(name string) *sgrdom.SecurityGroupRule {
	return &sgrdom.SecurityGroupRule{RegionalMetadata: rMeta(name),
		Spec: sgrdom.SecurityGroupRuleSpec{Direction: "egress", Protocol: "tcp"}}
}

func newNic(name string) *nicdom.Nic {
	return &nicdom.Nic{RegionalMetadata: rMeta(name),
		Spec: nicdom.NicSpec{Addresses: []string{"10.30.1.10"}, SubnetRef: ref("subnets/" + subnetName),
			SecurityGroupRefs: []commondomain.Reference{ref("security-groups/" + sgName)},
			PublicIpRefs:      []commondomain.Reference{ref("public-ips/" + pipName)}}}
}

func newInstance(name, sshKey string) *instdom.Instance {
	return &instdom.Instance{RegionalMetadata: rMeta(name),
		Spec: instdom.InstanceSpec{
			PrimaryNicRef: ptr(ref("nics/" + nicName)),
			BootVolume:    instdom.VolumeReference{DeviceRef: ref("block-storages/" + bootName)},
			SkuRef:        ref("skus/compute-sku-1"),
			SshKeys:       []string{sshKey},
			Zone:          "ITBG-1"}}
}

// ---- assertions ------------------------------------------------------------

// mustActive creates obj (idempotently) and waits until the SECA resource reports Active. A terminal
// Error state fails fast. This asserts the delegator reconciled it AND, for aruba-backed resources,
// that the operator provisioned the backing arubacloud.com CR - so it requires a real Aruba account.
func mustActive[T persistence.IdentifiableResource](t *testing.T, r *k8sadapter.RepoAdapter[T],
	obj T, timeout time.Duration, state func() (commondomain.ResourceState, bool)) {
	t.Helper()
	_, err := r.Create(ctx, obj)
	require.Truef(t, err == nil || kerrors.IsAlreadyExists(err), "create: %v", err)

	last := commondomain.ResourceState("<none>")
	err = wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true, func(context.Context) (bool, error) {
		s, ok := state()
		if !ok {
			return false, nil
		}
		last = s
		if s == commondomain.ResourceStateError {
			return false, fmt.Errorf("resource reached Error state")
		}
		return s == commondomain.ResourceStateActive, nil
	})
	require.NoErrorf(t, err, "should reach Active (last observed state: %s)", last)
}

// requireArubaActive asserts the plugin materialised the named arubacloud.com CR and the operator
// drove it Active.
func requireArubaActive(t *testing.T, resource, name, ns string) {
	t.Helper()
	require.Equalf(t, "Active", arubaPhase(resource, name, ns),
		"arubacloud.com %s/%s should be Active", resource, name)
}

// requireNoArubaCR asserts a no-op SECA resource created no backing arubacloud.com CR.
func requireNoArubaCR(t *testing.T, resource, name, ns string) {
	t.Helper()
	require.Equalf(t, "NOTFOUND", arubaPhase(resource, name, ns),
		"no-op %s should not materialise an arubacloud.com CR", resource)
}

// deleteFlowResources removes every SECA resource the flow creates, in reverse dependency order.
// Best-effort and non-blocking: it triggers the deletes (which cascade to the arubacloud.com CRs);
// the operator finishes the real Aruba teardown asynchronously. Registered via t.Cleanup so a failed
// run does not leak real cloud resources.
func deleteFlowResources() {
	_ = instRepo.Delete(ctx, newInstance(instName, ""))
	_ = nicRepo.Delete(ctx, newNic(nicName))
	_ = sgrRepo.Delete(ctx, newSecurityGroupRule(sgrName))
	_ = sgRepo.Delete(ctx, newSecurityGroup(sgName))
	_ = pipRepo.Delete(ctx, newPublicIp(pipName))
	_ = subRepo.Delete(ctx, newSubnet(subnetName))
	_ = rtRepo.Delete(ctx, newRouteTable(rtName))
	_ = netRepo.Delete(ctx, newNetwork(network))
	_ = igwRepo.Delete(ctx, newInternetGateway(igwName))
	_ = bsRepo.Delete(ctx, newBlockStorage(bootName))
	_ = wsRepo.Delete(ctx, newWorkspace(workspace))
}
