package handler

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backend "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	instancek8s "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
	computeskudom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/sku"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	sgdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	sgrdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"

	adaptconverter "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/skumap"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/repository"
	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
)

// Ensure ComputeInstanceHandler implements the Instance interface.
var _ instancek8s.InstancePlugin = (*ComputeInstanceHandler)(nil)

// ComputeInstanceHandler maps a SECA compute Instance to an Aruba CloudServer. A CloudServer needs
// a VPC, subnets, security groups, a key pair and a boot volume, none of which a SECA Instance
// names directly: they come from its NICs (subnet, security groups, public IPs), its ssh keys and
// its boot/data volumes. This handler resolves that graph and materialises what Aruba requires but
// SECA does not model as its own resource - the security groups (per VPC, at attach time) and the
// key pair (from the inline ssh key). See csp/aruba/README.md.
//
// Missing dependencies (a NIC not created yet, a subnet not active, no ssh key, no security group)
// gate the create with backend.ErrStillProcessing: the instance stays in "creating" and is retried,
// matching the other Aruba handlers. Aruba's CloudServer CRD carries no power field, so PowerOn and
// PowerOff are no-ops.
type ComputeInstanceHandler struct {
	wsRepository         persistence.ReaderRepo[*wsdom.Workspace]
	nicRepository        persistence.ReaderRepo[*nicdom.Nic]
	sgRepository         persistence.ReaderRepo[*sgdom.SecurityGroup]
	sgrRepository        persistence.ReaderRepo[*sgrdom.SecurityGroupRule]
	computeSkuRepository persistence.ReaderRepo[*computeskudom.InstanceSKU]

	prjRepository      repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList]
	subnetRepository   repository.Repository[*v1alpha1.Subnet, *v1alpha1.SubnetList]
	keyPairRepository  repository.Repository[*v1alpha1.KeyPair, *v1alpha1.KeyPairList]
	secGroupRepository repository.Repository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList]
	secRuleRepository  repository.Repository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList]
	blockStorageRepo   repository.Repository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList]
	elasticIPRepo      repository.Repository[*v1alpha1.ElasticIP, *v1alpha1.ElasticIPList]
	cloudServerRepo    repository.Repository[*v1alpha1.CloudServer, *v1alpha1.CloudServerList]
}

// NewComputeInstanceHandler wires the repositories the instance handler needs to resolve a SECA
// Instance's dependency graph and materialise the Aruba resources a CloudServer requires.
func NewComputeInstanceHandler(
	wsRepo persistence.ReaderRepo[*wsdom.Workspace],
	nicRepo persistence.ReaderRepo[*nicdom.Nic],
	sgRepo persistence.ReaderRepo[*sgdom.SecurityGroup],
	sgrRepo persistence.ReaderRepo[*sgrdom.SecurityGroupRule],
	computeSkuRepo persistence.ReaderRepo[*computeskudom.InstanceSKU],
	prjRepo repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList],
	subnetRepo repository.Repository[*v1alpha1.Subnet, *v1alpha1.SubnetList],
	keyPairRepo repository.Repository[*v1alpha1.KeyPair, *v1alpha1.KeyPairList],
	secGroupRepo repository.Repository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList],
	secRuleRepo repository.Repository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList],
	blockStorageRepo repository.Repository[*v1alpha1.BlockStorage, *v1alpha1.BlockStorageList],
	elasticIPRepo repository.Repository[*v1alpha1.ElasticIP, *v1alpha1.ElasticIPList],
	cloudServerRepo repository.Repository[*v1alpha1.CloudServer, *v1alpha1.CloudServerList],
) *ComputeInstanceHandler {
	return &ComputeInstanceHandler{
		wsRepository:         wsRepo,
		nicRepository:        nicRepo,
		sgRepository:         sgRepo,
		sgrRepository:        sgrRepo,
		computeSkuRepository: computeSkuRepo,
		prjRepository:        prjRepo,
		subnetRepository:     subnetRepo,
		keyPairRepository:    keyPairRepo,
		secGroupRepository:   secGroupRepo,
		secRuleRepository:    secRuleRepo,
		blockStorageRepo:     blockStorageRepo,
		elasticIPRepo:        elasticIPRepo,
		cloudServerRepo:      cloudServerRepo,
	}
}

// Create resolves the instance's dependency graph, materialises the key pair and security groups it
// needs, and creates the Aruba CloudServer. It is idempotent: every pass re-issues the creates and
// reports backend.ErrStillProcessing until the CloudServer is active.
func (h *ComputeInstanceHandler) Create(ctx context.Context, domain *instancedom.Instance) error {
	refs, err := h.resolve(ctx, domain)
	if err != nil {
		return err
	}

	cloudServer := adaptconverter.BuildCloudServer(domain, *refs)
	if err := h.cloudServerRepo.Create(ctx, cloudServer); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	observed := cloudServer.DeepCopy()
	if err := h.cloudServerRepo.Load(ctx, observed); err != nil {
		if apierrors.IsNotFound(err) {
			return backend.ErrStillProcessing
		}
		return err
	}
	if observed.Status.Phase != v1alpha1.ResourcePhaseActive {
		return backend.ErrStillProcessing
	}
	return nil
}

// Delete removes the CloudServer and, once it is gone, the key pair the instance owned. The
// materialised security groups are left in place: they may be shared with other instances, so they
// are reaped by the SecurityGroupHandler when the SECA security group itself is deleted, not here
// (see security-group.go / csp/aruba/README.md).
func (h *ComputeInstanceHandler) Delete(ctx context.Context, domain *instancedom.Instance) error {
	namespace := k8sadapter.ComputeNamespace(domain)

	cloudServer := &v1alpha1.CloudServer{
		ObjectMeta: metav1.ObjectMeta{Name: domain.Name, Namespace: namespace},
	}
	if err := h.cloudServerRepo.Delete(ctx, cloudServer); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	if err := h.cloudServerRepo.Load(ctx, cloudServer.DeepCopy()); err == nil {
		return backend.ErrStillProcessing // CloudServer still present, deletion in progress
	} else if !apierrors.IsNotFound(err) {
		return err
	}

	keyPair := &v1alpha1.KeyPair{
		ObjectMeta: metav1.ObjectMeta{Name: domain.Name + adaptconverter.KeyPairSuffix, Namespace: namespace},
	}
	if err := h.keyPairRepository.Delete(ctx, keyPair); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// PowerOn is a no-op: Aruba's CloudServer CRD exposes no power state, so power is tracked only on
// the SECA side.
func (h *ComputeInstanceHandler) PowerOn(_ context.Context, _ *instancedom.Instance) error {
	return nil
}

// PowerOff is a no-op for the same reason as PowerOn.
func (h *ComputeInstanceHandler) PowerOff(_ context.Context, _ *instancedom.Instance) error {
	return nil
}

// resolve gathers every reference a CloudServer needs, gating with backend.ErrStillProcessing while
// a dependency is missing and materialising the key pair and security groups along the way.
func (h *ComputeInstanceHandler) resolve(ctx context.Context, domain *instancedom.Instance) (*adaptconverter.CloudServerRefs, error) {
	tenant := domain.GetTenant()
	workspace := domain.GetWorkspace()
	prjNamespace := k8sadapter.ComputeNamespace(&res.Scope{Tenant: tenant})

	refs := &adaptconverter.CloudServerRefs{
		ProjectReference: v1alpha1.ResourceReference{Name: workspace, Namespace: prjNamespace},
	}

	if _, err := loadActiveWorkspace(ctx, h.wsRepository, domain); err != nil {
		return nil, err
	}

	project := &v1alpha1.Project{ObjectMeta: metav1.ObjectMeta{Name: workspace, Namespace: prjNamespace}}
	if err := loadActiveProject(ctx, h.prjRepository, project); err != nil {
		return nil, err
	}

	if err := h.resolveNetworking(ctx, domain, refs); err != nil {
		return nil, err
	}

	if len(domain.Spec.SshKeys) == 0 {
		return nil, backend.ErrStillProcessing // KeyPairReference is required and SECA has no ssh key here
	}
	keyPair := adaptconverter.BuildKeyPair(domain, domain.Spec.SshKeys[0])
	if err := h.keyPairRepository.Create(ctx, keyPair); err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, err
	}
	if err := loadActiveAruba(ctx, h.keyPairRepository, keyPair.DeepCopy()); err != nil {
		return nil, err
	}
	refs.KeyPairReference = v1alpha1.ResourceReference{Name: keyPair.Name, Namespace: keyPair.Namespace}

	if err := h.resolveVolumes(ctx, domain, refs); err != nil {
		return nil, err
	}

	// The instance's SKU is a SECA InstanceSKU describing capacity (vCPU/RAM); Aruba needs a named
	// flavor. Load the SKU and map its capacity to the Aruba flavor. A missing SKU CR is a not-ready
	// gate (catalog still syncing); a capacity with no Aruba flavor is a real error.
	skuName := lastSegment(domain.Spec.SkuRef.Resource)
	if skuName == "" {
		return nil, backend.ErrStillProcessing // SkuRef is required
	}
	sku := &computeskudom.InstanceSKU{RegionalMetadata: commondomain.RegionalMetadata{
		CommonMetadata: commondomain.CommonMetadata{Name: skuName},
		Scope:          res.Scope{Tenant: tenant},
	}}
	if err := h.computeSkuRepository.Load(ctx, &sku); err != nil {
		return nil, backend.ErrStillProcessing // SKU catalog not ready yet
	}
	flavor, err := skumap.ComputeFlavor(sku.Spec.VCPU, sku.Spec.Ram)
	if err != nil {
		return nil, err // no Aruba flavor for this capacity - surfaces as an Error condition
	}
	refs.FlavorName = flavor

	return refs, nil
}

// resolveNetworking fills in the references a CloudServer takes from the instance's network graph.
// A SECA Instance names none of them: the NICs carry the subnets, security groups and public IPs,
// and the security groups Aruba needs per VPC are materialised here. Expects refs.ProjectReference
// to be set, since the materialised security groups are created under it.
func (h *ComputeInstanceHandler) resolveNetworking(ctx context.Context, domain *instancedom.Instance, refs *adaptconverter.CloudServerRefs) error {
	tenant := domain.GetTenant()
	workspace := domain.GetWorkspace()
	wsNamespace := k8sadapter.ComputeNamespace(domain)

	subnetNames, sgNames, pipNames, err := h.resolveNics(ctx, domain)
	if err != nil {
		return err
	}
	if domain.Spec.SecurityGroupRef != nil {
		sgNames = appendUnique(sgNames, lastSegment(domain.Spec.SecurityGroupRef.Resource))
	}
	if len(subnetNames) == 0 {
		return backend.ErrStillProcessing // an Aruba CloudServer needs at least one subnet
	}

	// All of an instance's subnets live in one network's VPC; the first subnet fixes the VPC and
	// the network name used to name the materialised security groups.
	refs.SubnetReferences = make([]v1alpha1.ResourceReference, 0, len(subnetNames))
	var network string
	for i, resource := range subnetNames {
		subnet, err := h.resolveSubnet(ctx, tenant, workspace, resource)
		if err != nil {
			return err
		}
		refs.SubnetReferences = append(refs.SubnetReferences, v1alpha1.ResourceReference{Name: subnet.Name, Namespace: subnet.Namespace})
		if i == 0 {
			refs.VPCReference = subnet.Spec.VPCReference
			network = subnet.Labels[adaptconverter.LabelSubnetNetwork]
		}
	}

	if len(sgNames) == 0 {
		return backend.ErrStillProcessing // an Aruba CloudServer needs at least one security group
	}
	refs.SecurityGroupReferences = make([]v1alpha1.ResourceReference, 0, len(sgNames))
	for _, name := range sgNames {
		ref, err := h.materializeSecurityGroup(ctx, tenant, workspace, network, name, wsNamespace, domain.Region, refs.VPCReference, refs.ProjectReference)
		if err != nil {
			return err
		}
		refs.SecurityGroupReferences = append(refs.SecurityGroupReferences, ref)
	}

	if len(pipNames) > 0 {
		elasticIP := &v1alpha1.ElasticIP{
			ObjectMeta: metav1.ObjectMeta{Name: pipNames[0], Namespace: wsNamespace},
		}
		if err := loadActiveAruba(ctx, h.elasticIPRepo, elasticIP); err != nil {
			return err
		}
		refs.ElasticIPReference = &v1alpha1.ResourceReference{Name: pipNames[0], Namespace: wsNamespace}
	}
	return nil
}

// resolveVolumes fills in the boot and data volume references, plus the zone they pin the server to.
// The boot volume must be provisioned before the server, and Aruba requires the two to share a zone.
// SECA models no per-volume zone, so the volume's zone is the one that actually exists: take it from
// there rather than from the instance, and refuse a conflicting instance zone instead of sending a
// request Aruba would reject.
func (h *ComputeInstanceHandler) resolveVolumes(ctx context.Context, domain *instancedom.Instance, refs *adaptconverter.CloudServerRefs) error {
	wsNamespace := k8sadapter.ComputeNamespace(domain)

	bootName := lastSegment(domain.Spec.BootVolume.DeviceRef.Resource)
	if bootName == "" {
		return backend.ErrStillProcessing // BootVolumeReference is required
	}
	bootVolume := &v1alpha1.BlockStorage{
		ObjectMeta: metav1.ObjectMeta{Name: bootName, Namespace: wsNamespace},
	}
	if err := loadActiveAruba(ctx, h.blockStorageRepo, bootVolume); err != nil {
		return err
	}
	if zone := domain.Spec.Zone; zone != "" && zone != bootVolume.Spec.Zone {
		return fmt.Errorf("instance zone %q conflicts with boot volume %q in zone %q: an Aruba CloudServer must share a zone with its boot volume",
			zone, bootName, bootVolume.Spec.Zone)
	}
	refs.Zone = bootVolume.Spec.Zone
	refs.BootVolumeReference = v1alpha1.ResourceReference{Name: bootName, Namespace: wsNamespace}

	refs.DataVolumeReferences = []v1alpha1.ResourceReference{}
	for _, dv := range domain.Spec.DataVolumes {
		name := lastSegment(dv.DeviceRef.Resource)
		if name == "" {
			continue
		}
		dataVolume := &v1alpha1.BlockStorage{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: wsNamespace},
		}
		if err := loadActiveAruba(ctx, h.blockStorageRepo, dataVolume); err != nil {
			return err
		}
		refs.DataVolumeReferences = append(refs.DataVolumeReferences, v1alpha1.ResourceReference{Name: name, Namespace: wsNamespace})
	}
	return nil
}

// resolveNics loads the instance's NICs and collects the subnet references and the security group
// and public IP names they carry. A NIC that is not present yet gates the whole reconcile.
func (h *ComputeInstanceHandler) resolveNics(ctx context.Context, domain *instancedom.Instance) (subnets, secGroups, publicIps []string, err error) {
	refs := domain.Spec.AdditionalNicRefs
	if domain.Spec.PrimaryNicRef != nil {
		refs = append([]commondomain.Reference{*domain.Spec.PrimaryNicRef}, refs...)
	}

	for _, ref := range refs {
		nic := &nicdom.Nic{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: lastSegment(ref.Resource)},
				Scope:          res.Scope{Tenant: domain.GetTenant(), Workspace: domain.GetWorkspace()},
			},
		}
		if err := h.nicRepository.Load(ctx, &nic); err != nil {
			return nil, nil, nil, backend.ErrStillProcessing // NIC not created yet
		}

		// Kept whole rather than reduced to a name: a subnet reference may name the network it
		// is scoped under, which is what resolveSubnet needs to pick between same-named subnets.
		subnets = appendUnique(subnets, nic.Spec.SubnetRef.Resource)
		for _, sg := range nic.Spec.SecurityGroupRefs {
			secGroups = appendUnique(secGroups, lastSegment(sg.Resource))
		}
		for _, pip := range nic.Spec.PublicIpRefs {
			publicIps = appendUnique(publicIps, lastSegment(pip.Resource))
		}
	}
	return subnets, secGroups, publicIps, nil
}

// resolveSubnet finds the active Aruba Subnet that backs a SECA subnet reference.
//
// A SECA subnet is scoped under a network, so one workspace may hold several subnets of the same
// name in different networks. A reference that names its network ("networks/<network>/subnets/
// <name>") identifies one of them exactly. A bare "subnets/<name>" does not, and since the list
// order is not guaranteed, taking whichever came back first would wire the instance, its subnet
// reference and its materialised security groups into an arbitrary VPC - silently, and differently
// between reconciles. Refuse instead, and say how to disambiguate (see csp/aruba/README.md).
func (h *ComputeInstanceHandler) resolveSubnet(ctx context.Context, tenant, workspace, resource string) (*v1alpha1.Subnet, error) {
	list, err := h.subnetRepository.List(ctx, client.MatchingLabels{
		adaptconverter.LabelSubnetTenant:    tenant,
		adaptconverter.LabelSubnetWorkspace: workspace,
	})
	if err != nil {
		return nil, err
	}

	name, wantNetwork := lastSegment(resource), referenceNetwork(resource)

	var match *v1alpha1.Subnet
	for i := range list.Items {
		subnet := &list.Items[i]
		if subnet.Name != name || subnet.Status.Phase != v1alpha1.ResourcePhaseActive {
			continue
		}
		if wantNetwork != "" {
			if subnet.Labels[adaptconverter.LabelSubnetNetwork] == wantNetwork {
				return subnet, nil
			}
			continue
		}
		if match != nil && match.Labels[adaptconverter.LabelSubnetNetwork] != subnet.Labels[adaptconverter.LabelSubnetNetwork] {
			return nil, fmt.Errorf("subnet %q exists in more than one network (%q and %q): reference it as networks/<network>/subnets/%s",
				name, match.Labels[adaptconverter.LabelSubnetNetwork], subnet.Labels[adaptconverter.LabelSubnetNetwork], name)
		}
		match = subnet
	}

	if match == nil {
		return nil, backend.ErrStillProcessing // subnet not created or not active yet
	}
	return match, nil
}

// referenceNetwork returns the network segment of a SECA reference to a network-scoped resource.
// A subnet lives under a network, so a fully-qualified reference is
// "networks/<network>/subnets/<name>"; a bare "subnets/<name>" names no network and yields "".
func referenceNetwork(resource string) string {
	parts := strings.Split(resource, "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "networks" {
			return parts[i+1]
		}
	}
	return ""
}

// materializeSecurityGroup creates the Aruba SecurityGroup (and its rules) that backs a SECA
// security group inside the instance's VPC, returning a reference to it. Creates are idempotent so
// re-issuing on every pass is safe; the SECA security group must exist for its rules to be read.
func (h *ComputeInstanceHandler) materializeSecurityGroup(ctx context.Context, tenant, workspace, network, secaName, namespace, region string, vpcRef, projectRef v1alpha1.ResourceReference) (v1alpha1.ResourceReference, error) {
	seca := &sgdom.SecurityGroup{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: secaName},
			Scope:          res.Scope{Tenant: tenant, Workspace: workspace},
		},
	}
	if err := h.sgRepository.Load(ctx, &seca); err != nil {
		return v1alpha1.ResourceReference{}, backend.ErrStillProcessing // SECA security group not created yet
	}

	arubaSG := adaptconverter.BuildSecurityGroup(secaName, network, region, tenant, namespace, seca.Labels, vpcRef, projectRef)
	if err := h.secGroupRepository.Create(ctx, arubaSG); err != nil && !apierrors.IsAlreadyExists(err) {
		return v1alpha1.ResourceReference{}, err
	}

	rules := adaptconverter.NormalizeInlineRules(seca.Spec.Rules, seca.Labels)
	for _, rr := range seca.Spec.RuleRefs {
		standalone := &sgrdom.SecurityGroupRule{
			RegionalMetadata: commondomain.RegionalMetadata{
				CommonMetadata: commondomain.CommonMetadata{Name: lastSegment(rr.Resource)},
				Scope:          res.Scope{Tenant: tenant, Workspace: workspace},
			},
		}
		if err := h.sgrRepository.Load(ctx, &standalone); err != nil {
			return v1alpha1.ResourceReference{}, backend.ErrStillProcessing // referenced rule not created yet
		}
		rules = append(rules, adaptconverter.NormalizeStandaloneRule(standalone.Spec, standalone.Labels))
	}

	for _, rule := range adaptconverter.BuildSecurityRules(rules, arubaSG.Name, region, tenant, namespace, vpcRef, projectRef) {
		if err := h.secRuleRepository.Create(ctx, rule); err != nil && !apierrors.IsAlreadyExists(err) {
			return v1alpha1.ResourceReference{}, err
		}
	}

	// The CloudServer references this security group; Aruba rejects a server create (semantic 400)
	// whose security group is not yet provisioned in the CMP. Gate on the materialised SG being
	// active before handing back its reference, as we already do for the subnet.
	observed := arubaSG.DeepCopy()
	if err := h.secGroupRepository.Load(ctx, observed); err != nil {
		if apierrors.IsNotFound(err) {
			return v1alpha1.ResourceReference{}, backend.ErrStillProcessing
		}
		return v1alpha1.ResourceReference{}, err
	}
	if observed.Status.Phase != v1alpha1.ResourcePhaseActive {
		return v1alpha1.ResourceReference{}, backend.ErrStillProcessing
	}

	return v1alpha1.ResourceReference{Name: arubaSG.Name, Namespace: namespace}, nil
}

// lastSegment returns the name portion of a SECA reference resource ("<type>/<name>" -> "<name>").
func lastSegment(resource string) string {
	if i := strings.LastIndex(resource, "/"); i != -1 {
		return resource[i+1:]
	}
	return resource
}

// appendUnique appends v to s unless it is empty or already present.
func appendUnique(s []string, v string) []string {
	if v == "" {
		return s
	}
	if slices.Contains(s, v) {
		return s
	}
	return append(s, v)
}
