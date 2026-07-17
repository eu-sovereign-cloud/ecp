package crossplane

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	commonbackend "github.com/eu-sovereign-cloud/ecp/resource/common/backend"
	skuk8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku/backend/kubernetes"
	nick8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	subnetk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
	bsk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgk8s "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
)

// readSKU reads an InstanceSKU CR and returns IONOS cores + RAM (MB).
func (c *base) readSKU(ctx context.Context, ns, name string) (cores, ramMB float64, err error) {
	sku := &skuk8s.InstanceSKU{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: name}, sku); err != nil {
		return 0, 0, fmt.Errorf("read instance sku %q: %w", name, err)
	}
	return float64(sku.Spec.VCPU), float64(sku.Spec.Ram) * 1024, nil
}

// readBootImageAlias resolves the IONOS image alias + size for an image-backed block storage.
func (c *base) readBootImageAlias(ctx context.Context, ns, bsName string) (alias string, sizeGB int, err error) {
	bsCR := &bsk8s.BlockStorage{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: bsName}, bsCR); err != nil {
		return "", 0, fmt.Errorf("read block storage %q: %w", bsName, err)
	}
	bs, err := bsk8s.BlockStorageFromCR(bsCR)
	if err != nil {
		return "", 0, fmt.Errorf("convert block storage %q: %w", bsName, err)
	}
	if bs.Spec.SourceImageRef == nil {
		return "", 0, fmt.Errorf("block storage %q has no source image", bsName)
	}

	imageName := commonbackend.ParseReference(*bs.Spec.SourceImageRef, "").Name
	imgCR := &imgk8s.Image{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: imageName}, imgCR); err != nil {
		return "", 0, fmt.Errorf("read image %q: %w", imageName, err)
	}
	img, err := imgk8s.ImageFromCR(imgCR)
	if err != nil {
		return "", 0, fmt.Errorf("convert image %q: %w", imageName, err)
	}
	alias, err = translateImage(img.Labels["base"], img.Labels["version"])
	if err != nil {
		return "", 0, err
	}
	return alias, bs.Spec.SizeGB, nil
}

// readNicNetworking resolves the LAN name (via subnet -> network) and the reserved
// public IP (via the NIC's public-ip ref -> IPBlock) for a NIC.
func (c *base) readNicNetworking(ctx context.Context, ns, nicName string) (lanName, publicIP string, err error) {
	nicCR := &nick8s.NIC{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: nicName}, nicCR); err != nil {
		return "", "", fmt.Errorf("read nic %q: %w", nicName, err)
	}
	nic, err := nick8s.NicFromCR(nicCR)
	if err != nil {
		return "", "", fmt.Errorf("convert nic %q: %w", nicName, err)
	}

	subnetName := commonbackend.ParseReference(nic.Spec.SubnetRef, "").Name
	subnetCR := &subnetk8s.Subnet{}
	if err := c.client.Get(ctx, client.ObjectKey{Namespace: ns, Name: subnetName}, subnetCR); err != nil {
		return "", "", fmt.Errorf("read subnet %q: %w", subnetName, err)
	}
	subnet, err := subnetk8s.SubnetFromCR(subnetCR)
	if err != nil {
		return "", "", fmt.Errorf("convert subnet %q: %w", subnetName, err)
	}
	lanName = subnet.GetNetwork()

	if len(nic.Spec.PublicIpRefs) == 0 {
		return "", "", fmt.Errorf("nic %q has no public ip ref", nicName)
	}
	publicIPName := commonbackend.ParseReference(nic.Spec.PublicIpRefs[0], "").Name
	publicIP, err = readReservedIP(ctx, c.client, ns, publicIPName)
	if err != nil {
		return "", "", err
	}
	return lanName, publicIP, nil
}
