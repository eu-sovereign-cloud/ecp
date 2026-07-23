package handler

import (
	"context"
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
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	sgdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	sgrdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"

	adaptconverter "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/converter"
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
	wsRepository  persistence.ReaderRepo[*wsdom.Workspace]
	nicRepository persistence.ReaderRepo[*nicdom.Nic]
	sgRepository  persistence.ReaderRepo[*sgdom.SecurityGroup]
	sgrRepository persistence.ReaderRepo[*sgrdom.SecurityGroupRule]

	prjRepository      repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList]
	subnetRepository   repository.Repository[*v1alpha1.Subnet, *v1alpha1.SubnetList]
	keyPairRepository  repository.Repository[*v1alpha1.KeyPair, *v1alpha1.KeyPairList]
	secGroupRepository repository.Repository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList]
	secRuleRepository  repository.Repository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList]
	cloudServerRepo    repository.Repository[*v1alpha1.CloudServer, *v1alpha1.CloudServerList]
}

// NewComputeInstanceHandler wires the repositories the instance handler needs to resolve a SECA
// Instance's dependency graph and materialise the Aruba resources a CloudServer requires.
func NewComputeInstanceHandler(
	wsRepo persistence.ReaderRepo[*wsdom.Workspace],
	nicRepo persistence.ReaderRepo[*nicdom.Nic],
	sgRepo persistence.ReaderRepo[*sgdom.SecurityGroup],
	sgrRepo persistence.ReaderRepo[*sgrdom.SecurityGroupRule],
	prjRepo repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList],
	subnetRepo repository.Repository[*v1alpha1.Subnet, *v1alpha1.SubnetList],
	keyPairRepo repository.Repository[*v1alpha1.KeyPair, *v1alpha1.KeyPairList],
	secGroupRepo repository.Repository[*v1alpha1.SecurityGroup, *v1alpha1.SecurityGroupList],
	secRuleRepo repository.Repository[*v1alpha1.SecurityRule, *v1alpha1.SecurityRuleList],
	cloudServerRepo repository.Repository[*v1alpha1.CloudServer, *v1alpha1.CloudServerList],
) *ComputeInstanceHandler {
	return &ComputeInstanceHandler{
		wsRepository:       wsRepo,
		nicRepository:      nicRepo,
		sgRepository:       sgRepo,
		sgrRepository:      sgrRepo,
		prjRepository:      prjRepo,
		subnetRepository:   subnetRepo,
		keyPairRepository:  keyPairRepo,
		secGroupRepository: secGroupRepo,
		secRuleRepository:  secRuleRepo,
		cloudServerRepo:    cloudServerRepo,
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
// materialised security groups are left in place: they may be shared with other instances and do
// not interfere once the SECA security group still exists (see csp/aruba/README.md).
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
	wsNamespace := k8sadapter.ComputeNamespace(domain)
	prjNamespace := k8sadapter.ComputeNamespace(&res.Scope{Tenant: tenant})
	projectRef := v1alpha1.ResourceReference{Name: workspace, Namespace: prjNamespace}

	if _, err := loadActiveWorkspace(ctx, h.wsRepository, domain); err != nil {
		return nil, err
	}

	project := &v1alpha1.Project{ObjectMeta: metav1.ObjectMeta{Name: workspace, Namespace: prjNamespace}}
	if err := loadActiveProject(ctx, h.prjRepository, project); err != nil {
		return nil, err
	}

	subnetNames, sgNames, pipNames, err := h.resolveNics(ctx, domain)
	if err != nil {
		return nil, err
	}
	if domain.Spec.SecurityGroupRef != nil {
		sgNames = appendUnique(sgNames, lastSegment(domain.Spec.SecurityGroupRef.Resource))
	}
	if len(subnetNames) == 0 {
		return nil, backend.ErrStillProcessing // an Aruba CloudServer needs at least one subnet
	}

	// All of an instance's subnets live in one network's VPC; the first subnet fixes the VPC and
	// the network name used to name the materialised security groups.
	subnetRefs := make([]v1alpha1.ResourceReference, 0, len(subnetNames))
	var vpcRef v1alpha1.ResourceReference
	var network string
	for i, name := range subnetNames {
		subnet, err := h.resolveSubnet(ctx, tenant, workspace, name)
		if err != nil {
			return nil, err
		}
		subnetRefs = append(subnetRefs, v1alpha1.ResourceReference{Name: subnet.Name, Namespace: subnet.Namespace})
		if i == 0 {
			vpcRef = subnet.Spec.VPCReference
			network = subnet.Labels["seca.subnet/network"]
		}
	}

	if len(sgNames) == 0 {
		return nil, backend.ErrStillProcessing // an Aruba CloudServer needs at least one security group
	}
	sgRefs := make([]v1alpha1.ResourceReference, 0, len(sgNames))
	for _, name := range sgNames {
		ref, err := h.materializeSecurityGroup(ctx, tenant, workspace, network, name, wsNamespace, domain.Region, vpcRef, projectRef)
		if err != nil {
			return nil, err
		}
		sgRefs = append(sgRefs, ref)
	}

	if len(domain.Spec.SshKeys) == 0 {
		return nil, backend.ErrStillProcessing // KeyPairReference is required and SECA has no ssh key here
	}
	keyPair := adaptconverter.BuildKeyPair(domain, domain.Spec.SshKeys[0])
	if err := h.keyPairRepository.Create(ctx, keyPair); err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, err
	}

	bootName := lastSegment(domain.Spec.BootVolume.DeviceRef.Resource)
	if bootName == "" {
		return nil, backend.ErrStillProcessing // BootVolumeReference is required
	}
	var dataRefs []v1alpha1.ResourceReference
	for _, dv := range domain.Spec.DataVolumes {
		if name := lastSegment(dv.DeviceRef.Resource); name != "" {
			dataRefs = append(dataRefs, v1alpha1.ResourceReference{Name: name, Namespace: wsNamespace})
		}
	}

	flavor := lastSegment(domain.Spec.SkuRef.Resource)
	if flavor == "" {
		return nil, backend.ErrStillProcessing // FlavorName is required
	}

	var eipRef *v1alpha1.ResourceReference
	if len(pipNames) > 0 {
		eipRef = &v1alpha1.ResourceReference{Name: pipNames[0], Namespace: wsNamespace}
	}

	return &adaptconverter.CloudServerRefs{
		FlavorName:              flavor,
		VPCReference:            vpcRef,
		SubnetReferences:        subnetRefs,
		SecurityGroupReferences: sgRefs,
		KeyPairReference:        v1alpha1.ResourceReference{Name: keyPair.Name, Namespace: keyPair.Namespace},
		BootVolumeReference:     v1alpha1.ResourceReference{Name: bootName, Namespace: wsNamespace},
		DataVolumeReferences:    dataRefs,
		ElasticIPReference:      eipRef,
		ProjectReference:        projectRef,
	}, nil
}

// resolveNics loads the instance's NICs and collects the subnet, security group and public IP names
// they reference. A NIC that is not present yet gates the whole reconcile.
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

		subnets = appendUnique(subnets, lastSegment(nic.Spec.SubnetRef.Resource))
		for _, sg := range nic.Spec.SecurityGroupRefs {
			secGroups = appendUnique(secGroups, lastSegment(sg.Resource))
		}
		for _, pip := range nic.Spec.PublicIpRefs {
			publicIps = appendUnique(publicIps, lastSegment(pip.Resource))
		}
	}
	return subnets, secGroups, publicIps, nil
}

// resolveSubnet finds the active Aruba Subnet that backs a SECA subnet name. SECA NIC references
// carry no network, so the subnet is located by workspace label across namespaces and matched by
// name; the first active match wins (see csp/aruba/README.md).
func (h *ComputeInstanceHandler) resolveSubnet(ctx context.Context, tenant, workspace, name string) (*v1alpha1.Subnet, error) {
	list, err := h.subnetRepository.List(ctx, client.MatchingLabels{
		"seca.subnet/tenant":    tenant,
		"seca.subnet/workspace": workspace,
	})
	if err != nil {
		return nil, err
	}

	for i := range list.Items {
		subnet := &list.Items[i]
		if subnet.Name == name && subnet.Status.Phase == v1alpha1.ResourcePhaseActive {
			return subnet, nil
		}
	}
	return nil, backend.ErrStillProcessing // subnet not created or not active yet
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

	arubaSG := adaptconverter.BuildSecurityGroup(secaName, network, region, tenant, namespace, vpcRef, projectRef)
	if err := h.secGroupRepository.Create(ctx, arubaSG); err != nil && !apierrors.IsAlreadyExists(err) {
		return v1alpha1.ResourceReference{}, err
	}

	rules := adaptconverter.NormalizeInlineRules(seca.Spec.Rules)
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
		rules = append(rules, adaptconverter.NormalizeStandaloneRule(standalone.Spec))
	}

	for _, rule := range adaptconverter.BuildSecurityRules(rules, arubaSG.Name, region, tenant, namespace, vpcRef, projectRef) {
		if err := h.secRuleRepository.Create(ctx, rule); err != nil && !apierrors.IsAlreadyExists(err) {
			return v1alpha1.ResourceReference{}, err
		}
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
	for _, e := range s {
		if e == v {
			return s
		}
	}
	return append(s, v)
}
