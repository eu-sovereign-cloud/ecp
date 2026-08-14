package crossplane

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	"github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	skuk8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku/backend/kubernetes"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
)

// readSKU reads an InstanceSKU CR and returns IONOS cores + RAM (MB). The CR's
// namespace is derived from skuRef itself (via ParseReference), since the SECA
// gateway writes ECP CRs at hash(tenant/workspace), not hash(tenant) alone.
func (c *base) readSKU(ctx context.Context, skuRef domain.Reference, defaultTenant string) (cores, ramMB float64, err error) {
	t := commonbackend.ParseReference(skuRef, defaultTenant)
	ns := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: t.Tenant, Workspace: t.Workspace})

	sku := &skuk8s.InstanceSKU{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: t.Name}, sku); err != nil {
		return 0, 0, fmt.Errorf("read instance sku %q: %w", t.Name, err)
	}
	return float64(sku.Spec.VCPU), float64(sku.Spec.Ram) * 1024, nil
}

// readBootImageAlias resolves the IONOS image alias + size for an image-backed block
// storage. The block storage reference falls back to the instance's own
// tenant/workspace when it doesn't carry its own (same-workspace references omit
// both). The image it references is resolved independently, since images are
// tenant-scoped and the reference may carry its own tenant.
func (c *base) readBootImageAlias(ctx context.Context, bootRef domain.Reference, defaultTenant, defaultWorkspace string) (alias string, sizeGB int, err error) {
	t := commonbackend.ParseReference(bootRef, defaultTenant)
	if t.Workspace == "" {
		t.Workspace = defaultWorkspace
	}
	ns := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: t.Tenant, Workspace: t.Workspace})

	bsCR := &bsk8s.BlockStorage{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: t.Name}, bsCR); err != nil {
		return "", 0, fmt.Errorf("read block storage %q: %w", t.Name, err)
	}
	bs, err := bsk8s.BlockStorageFromCR(bsCR)
	if err != nil {
		return "", 0, fmt.Errorf("convert block storage %q: %w", t.Name, err)
	}
	if bs.Spec.SourceImageRef == nil {
		return "", 0, fmt.Errorf("block storage %q has no source image", t.Name)
	}

	imgTarget := commonbackend.ParseReference(*bs.Spec.SourceImageRef, t.Tenant)
	imgNs := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: imgTarget.Tenant, Workspace: imgTarget.Workspace})
	imgCR := &imgk8s.Image{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: imgNs, Name: imgTarget.Name}, imgCR); err != nil {
		return "", 0, fmt.Errorf("read image %q: %w", imgTarget.Name, err)
	}
	img, err := imgk8s.ImageFromCR(imgCR)
	if err != nil {
		return "", 0, fmt.Errorf("convert image %q: %w", imgTarget.Name, err)
	}
	// Prefer the reconstructed domain labels (gateway-written images carry base/version
	// as keyed labels), but fall back to raw CR metadata labels so images seeded directly
	// (e.g. kubectl-applied catalog images) with plain base/version labels also resolve.
	base, version := img.Labels["base"], img.Labels["version"]
	if base == "" || version == "" {
		raw := imgCR.GetLabels()
		if base == "" {
			base = raw["base"]
		}
		if version == "" {
			version = raw["version"]
		}
	}
	alias, err = translateImage(base, version)
	if err != nil {
		return "", 0, err
	}
	return alias, bs.Spec.SizeGB, nil
}

// networkFromSubnetRef extracts the owning network name from a subnet reference whose
// Resource path is ".../networks/<network>/subnets/<name>".
func networkFromSubnetRef(subnetRef domain.Reference) (string, error) {
	parts := strings.Split(subnetRef.Resource, "/")
	for i, p := range parts {
		if p == "networks" && i+1 < len(parts) {
			return parts[i+1], nil
		}
	}
	return "", fmt.Errorf("subnet reference %q has no network segment", subnetRef.Resource)
}

// readNicNetworking resolves the LAN name (from the NIC's subnet reference path) and
// the reserved public IP (via the NIC's public-ip ref -> IPBlock) for a NIC. The NIC
// reference falls back to the instance's own tenant/workspace when it doesn't carry
// its own (same-workspace references omit both).
func (c *base) readNicNetworking(ctx context.Context, nicRef domain.Reference, defaultTenant, defaultWorkspace string) (lanName, publicIP string, err error) {
	t := commonbackend.ParseReference(nicRef, defaultTenant)
	if t.Workspace == "" {
		t.Workspace = defaultWorkspace
	}
	ns := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: t.Tenant, Workspace: t.Workspace})

	nicCR := &nick8s.NIC{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: t.Name}, nicCR); err != nil {
		return "", "", fmt.Errorf("read nic %q: %w", t.Name, err)
	}
	nic, err := nick8s.NicFromCR(nicCR)
	if err != nil {
		return "", "", fmt.Errorf("convert nic %q: %w", t.Name, err)
	}

	lanName, err = networkFromSubnetRef(nic.Spec.SubnetRef)
	if err != nil {
		return "", "", err
	}

	// A NIC with no public-ip ref is not an error: callers may leave public_ip_ids
	// unset (as the POC does), so we fall back to IONOS auto-assigning a public
	// IPv4 via DHCP on the public LAN.
	if len(nic.Spec.PublicIpRefs) == 0 {
		return lanName, "", nil
	}

	// The IPBlock is a crossplane CR created by the PublicIP plugin at hash(tenant),
	// not at the ECP CR namespace-derivation scheme. Use the reference's own tenant
	// (which may differ from the instance's) rather than defaultTenant.
	ipTarget := commonbackend.ParseReference(nic.Spec.PublicIpRefs[0], defaultTenant)
	ipNs := k8sadapter.ComputeNamespace(&kernelresource.Scope{Tenant: ipTarget.Tenant})
	publicIP, err = readReservedIP(ctx, c.client, ipNs, ipTarget.Name)
	if err != nil {
		return "", "", err
	}
	return lanName, publicIP, nil
}
