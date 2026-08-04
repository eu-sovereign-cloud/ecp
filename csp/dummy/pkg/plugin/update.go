package plugin

import (
	"context"
	"log/slog"

	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	instanceconv "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
	internetgatewayconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	networkconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	nicconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
	publicipconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip/backend/kubernetes"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
	routetableconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	securitygroupruleconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
	securitygroupconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
	subnetconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	storageconv "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
	imageconv "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	workspaceconv "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

// AppliedLabelsAnnotation records the labels the plugin was last asked to apply. There is no cloud
// behind the dummy CSP, so "applying" an update means leaving evidence that the change reached the
// plugin - in a place both a test and an operator can read back through the API.
//
// A real CSP writes the labels onto the provider resource instead; see the Aruba plugin, which
// turns them into tags.
const AppliedLabelsAnnotation = "dummy.csp/applied-labels"

// applyUpdate records the resource's labels back onto the resource itself, through the same
// backing store the simulated create and delete write to.
func applyUpdate[D persistence.IdentifiableResource](
	ctx context.Context,
	resource D,
	annotations *map[string]string,
	labels map[string]string,
	gvr schema.GroupVersionResource,
	toCR kubernetesadapter.DomainToK8s[D],
	fromCR kubernetesadapter.K8sToDomain[D],
	logger *slog.Logger,
) error {
	return recordAppliedLabels(ctx, resource, annotations, labels, func(ctx context.Context, r D) error {
		dynamicClient, err := newDynamicClient()
		if err != nil {
			return err
		}

		if _, err := kubernetesadapter.NewRepoAdapter(dynamicClient, gvr, logger, toCR, fromCR).Update(ctx, r); err != nil {
			return err
		}

		logger.Info("dummy plugin: applied update", "resource_name", r.GetName(), "labels", formatLabels(labels))

		return nil
	})
}

// recordAppliedLabels stamps the labels onto the annotation and persists, but only when the record
// is stale.
//
// The conditional write is not an optimisation. Update runs on every reconcile of an active
// resource, and the controller reconciles on its own writes, so persisting unconditionally would
// spin the resource forever. Note that the short circuit comes first: an up-to-date resource never
// reaches the persist step at all, and so never builds a client.
func recordAppliedLabels[D persistence.IdentifiableResource](
	ctx context.Context,
	resource D,
	annotations *map[string]string,
	labels map[string]string,
	persist func(context.Context, D) error,
) error {
	applied := formatLabels(labels)
	if (*annotations)[AppliedLabelsAnnotation] == applied {
		return nil
	}

	if *annotations == nil {
		*annotations = make(map[string]string)
	}
	(*annotations)[AppliedLabelsAnnotation] = applied

	return persist(ctx, resource)
}

// formatLabels renders labels as a stable "k=v,k=v" string. Set.String sorts, which matters:
// Go map iteration order is random and an unstable rendering would look like a change on every pass.
func formatLabels(labels map[string]string) string {
	return k8slabels.Set(labels).String()
}

// Update is the same recording step for every dummy resource: there is no provider state to
// diverge from, so each one just reports the labels it was handed. They live together here
// rather than beside their Create and Delete because they are one behaviour, not twelve.

func (b *BlockStorage) Update(ctx context.Context, resource *bsdom.BlockStorage) error {
	return applyUpdate(ctx, resource, &resource.Annotations, resource.Labels,
		storageconv.BlockStorageGVR, storageconv.BlockStorageToCR, storageconv.BlockStorageFromCR, b.logger)
}

func (i *Image) Update(ctx context.Context, resource *imgdom.Image) error {
	return applyUpdate(ctx, resource, &resource.Annotations, resource.Labels,
		imageconv.ImageGVR, imageconv.ImageToCR, imageconv.ImageFromCR, i.logger)
}

func (i *Instance) Update(ctx context.Context, resource *instancedom.Instance) error {
	return applyUpdate(ctx, resource, &resource.Annotations, resource.Labels,
		instanceconv.InstanceGVR, instanceconv.InstanceToCR, instanceconv.InstanceFromCR, i.logger)
}

func (ig *InternetGateway) Update(ctx context.Context, resource *internetgatewaydom.InternetGateway) error {
	return applyUpdate(ctx, resource, &resource.Annotations, resource.Labels,
		internetgatewayconv.InternetGatewayGVR, internetgatewayconv.InternetGatewayToCR, internetgatewayconv.InternetGatewayFromCR, ig.logger)
}

func (n *Network) Update(ctx context.Context, resource *netdom.Network) error {
	return applyUpdate(ctx, resource, &resource.Annotations, resource.Labels,
		networkconv.NetworkGVR, networkconv.NetworkToCR, networkconv.NetworkFromCR, n.logger)
}

func (n *Nic) Update(ctx context.Context, resource *nicdom.Nic) error {
	return applyUpdate(ctx, resource, &resource.Annotations, resource.Labels,
		nicconv.NICGVR, nicconv.NicToCR, nicconv.NicFromCR, n.logger)
}

func (p *PublicIp) Update(ctx context.Context, resource *publicipdom.PublicIp) error {
	return applyUpdate(ctx, resource, &resource.Annotations, resource.Labels,
		publicipconv.PublicIPGVR, publicipconv.PublicIpToCR, publicipconv.PublicIpFromCR, p.logger)
}

func (rt *RouteTable) Update(ctx context.Context, resource *routetabledom.RouteTable) error {
	return applyUpdate(ctx, resource, &resource.Annotations, resource.Labels,
		routetableconv.RouteTableGVR, routetableconv.RouteTableToCR, routetableconv.RouteTableFromCR, rt.logger)
}

func (sg *SecurityGroup) Update(ctx context.Context, resource *securitygroupdom.SecurityGroup) error {
	return applyUpdate(ctx, resource, &resource.Annotations, resource.Labels,
		securitygroupconv.SecurityGroupGVR, securitygroupconv.SecurityGroupToCR, securitygroupconv.SecurityGroupFromCR, sg.logger)
}

func (sgr *SecurityGroupRule) Update(ctx context.Context, resource *securitygroupruledom.SecurityGroupRule) error {
	return applyUpdate(ctx, resource, &resource.Annotations, resource.Labels,
		securitygroupruleconv.SecurityGroupRuleGVR, securitygroupruleconv.SecurityGroupRuleToCR, securitygroupruleconv.SecurityGroupRuleFromCR, sgr.logger)
}

func (s *Subnet) Update(ctx context.Context, resource *subnetdom.Subnet) error {
	return applyUpdate(ctx, resource, &resource.Annotations, resource.Labels,
		subnetconv.SubnetGVR, subnetconv.SubnetToCR, subnetconv.SubnetFromCR, s.logger)
}

func (w *Workspace) Update(ctx context.Context, resource *wsdom.Workspace) error {
	return applyUpdate(ctx, resource, &resource.Annotations, resource.Labels,
		workspaceconv.WorkspaceGVR, workspaceconv.WorkspaceToCR, workspaceconv.WorkspaceFromCR, w.logger)
}
