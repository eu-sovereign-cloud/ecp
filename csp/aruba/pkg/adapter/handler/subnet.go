package handler

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backend "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
	subnetk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/delegated"
	mutator_bypass "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/mutator"
	resolver_bypass "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/resolver"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/converter"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/repository"
)

// Ensure SubnetHandler implements the Subnet interface
var _ subnetk8s.SubnetPlugin = (*SubnetHandler)(nil)

// SubnetHandler handles Subnet resources by interacting with Aruba Subnet.
// It is responsible for translating Subnet resources to Aruba Subnet
// and managing their lifecycle (Create/Delete).
type SubnetHandler struct {
	wsRepository     persistence.ReaderRepo[*wsdom.Workspace]
	subnetRepository repository.Repository[*v1alpha1.Subnet, *v1alpha1.SubnetList]
	vpcRepository    repository.Repository[*v1alpha1.VPC, *v1alpha1.VPCList]
	prjRepository    repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList]
	subnetConverter  converter.Converter[*subnetdom.Subnet, *v1alpha1.Subnet]
	wsConverter      converter.Converter[*wsdom.Workspace, *v1alpha1.Project]
	createDelegated  *delegated.GenericDelegated[*subnetdom.Subnet, *SecaSubnetBundle, *ArubaSubnetBundle]
	deleteDelegated  *delegated.GenericDelegated[*subnetdom.Subnet, *SecaSubnetBundle, *ArubaSubnetBundle]
}

type SecaSubnetBundle struct {
	Subnet    *subnetdom.Subnet
	Workspace *wsdom.Workspace
}

type ArubaSubnetBundle struct {
	Subnet  *v1alpha1.Subnet
	Project *v1alpha1.Project
}

// NewSubnetHandler creates a new SubnetHandler with the provided repositories and converters.
// It sets up the necessary delegated operations for creating and deleting Subnet resources.
// The handler uses bypass mutators since no mutation is needed on the Aruba Subnet objects.
func NewSubnetHandler(
	wsRepo persistence.ReaderRepo[*wsdom.Workspace],
	subnetRepo repository.Repository[*v1alpha1.Subnet, *v1alpha1.SubnetList],
	vpcRepo repository.Repository[*v1alpha1.VPC, *v1alpha1.VPCList],
	prjRepo repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList],
	subnetConv converter.Converter[*subnetdom.Subnet, *v1alpha1.Subnet],
	wsConv converter.Converter[*wsdom.Workspace, *v1alpha1.Project],
) *SubnetHandler {
	handler := &SubnetHandler{
		wsRepository:     wsRepo,
		subnetRepository: subnetRepo,
		vpcRepository:    vpcRepo,
		prjRepository:    prjRepo,
		subnetConverter:  subnetConv,
		wsConverter:      wsConv,
	}

	handler.createDelegated = delegated.NewDelegated(
		handler.resolveSecaSubnetDependencies,
		handler.FromSECABundleToAruba,
		handler.resolveArubaSubnetDependencies,
		mutator_bypass.BypassMutateFunc[*ArubaSubnetBundle, *SecaSubnetBundle],
		handler.propagateCreate,
		handler.checkSubnetCreated,
	)

	handler.deleteDelegated = delegated.NewDelegated(
		handler.BypassDependencyResolver,
		handler.FromSECABundleToAruba,
		resolver_bypass.BypassResolveDependenciesFunc[*ArubaSubnetBundle],
		mutator_bypass.BypassMutateFunc[*ArubaSubnetBundle, *SecaSubnetBundle],
		handler.propagateDelete,
		handler.checkSubnetDeleted,
	)

	return handler
}

// Create creates a new Subnet by creating an Aruba Subnet.
func (h *SubnetHandler) Create(ctx context.Context, domain *subnetdom.Subnet) error {
	return h.createDelegated.Do(ctx, domain)
}

// Delete deletes an existing Subnet by deleting the corresponding Aruba Subnet.
func (h *SubnetHandler) Delete(ctx context.Context, domain *subnetdom.Subnet) error {
	return h.deleteDelegated.Do(ctx, domain)
}

// checkSubnetCreated reports whether the Aruba Subnet already exists and has
// reached the active phase.
func (h *SubnetHandler) checkSubnetCreated(ctx context.Context, _ *SecaSubnetBundle, bundle *ArubaSubnetBundle) (bool, error) {
	observed := bundle.Subnet.DeepCopy()

	if err := h.subnetRepository.Load(ctx, observed); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil // Not created yet, it must be created.
		}

		return false, err
	}

	return observed.Status.Phase == v1alpha1.ResourcePhaseActive, nil
}

// checkSubnetDeleted reports whether the Aruba Subnet is gone.
func (h *SubnetHandler) checkSubnetDeleted(ctx context.Context, _ *SecaSubnetBundle, bundle *ArubaSubnetBundle) (bool, error) {
	observed := bundle.Subnet.DeepCopy()

	if err := h.subnetRepository.Load(ctx, observed); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil // Gone, deletion is complete.
		}

		return false, err
	}

	return false, nil // Still present, deletion is in progress.
}

func (h *SubnetHandler) BypassDependencyResolver(_ context.Context, domain *subnetdom.Subnet) (*SecaSubnetBundle, error) {
	return &SecaSubnetBundle{
		Subnet: domain,
	}, nil
}

// resolveSecaSubnetDependencies gates on the workspace being active. The owning network needs no
// check of its own here: the Aruba resolver waits for that network's VPC to be active, which can
// only happen once the SECA network itself progressed.
func (h *SubnetHandler) resolveSecaSubnetDependencies(ctx context.Context, domain *subnetdom.Subnet) (*SecaSubnetBundle, error) {
	ws, err := loadActiveWorkspace(ctx, h.wsRepository, domain)
	if err != nil {
		return nil, err
	}

	return &SecaSubnetBundle{
		Subnet:    domain,
		Workspace: ws,
	}, nil
}

// resolveArubaSubnetDependencies gates on the Project and the parent VPC being active. Aruba
// rejects a subnet whose VPC is still provisioning, so the wait is mandatory rather than cosmetic.
func (h *SubnetHandler) resolveArubaSubnetDependencies(ctx context.Context, arubaBundle *ArubaSubnetBundle) (*ArubaSubnetBundle, error) {
	if err := loadActiveProject(ctx, h.prjRepository, arubaBundle.Project); err != nil {
		return nil, err
	}

	// The converter already resolved which VPC owns this subnet; reuse that reference rather
	// than recomputing the network's namespace here.
	vpcRef := arubaBundle.Subnet.Spec.VPCReference
	vpc := &v1alpha1.VPC{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vpcRef.Name,
			Namespace: vpcRef.Namespace,
		},
	}

	if err := h.vpcRepository.Load(ctx, vpc); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, backend.ErrStillProcessing // VPC not found, wait for the network to create it
		}

		return nil, err
	}

	if vpc.Status.Phase != v1alpha1.ResourcePhaseActive {
		return nil, backend.ErrStillProcessing // VPC is not ready, wait for it to be active
	}

	return arubaBundle, nil
}

func (h *SubnetHandler) FromSECABundleToAruba(from *SecaSubnetBundle) (*ArubaSubnetBundle, error) {
	response := &ArubaSubnetBundle{}

	if from.Workspace != nil {
		prj, err := h.wsConverter.FromSECAToAruba(from.Workspace)
		if err != nil {
			return nil, err
		}

		response.Project = prj
	}

	subnet, err := h.subnetConverter.FromSECAToAruba(from.Subnet)
	if err != nil {
		return nil, err
	}

	response.Subnet = subnet

	return response, nil
}

// propagateCreate creates the Aruba Subnet. It is idempotent: because the create
// is (re)issued on every pass until the resource becomes active, an already
// existing resource is not treated as an error.
func (h *SubnetHandler) propagateCreate(ctx context.Context, from *ArubaSubnetBundle) error {
	if err := h.subnetRepository.Create(ctx, from.Subnet); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// propagateDelete deletes the Aruba Subnet. It is idempotent: because the delete
// is (re)issued on every pass until the resource is gone, an already missing
// resource is not treated as an error.
func (h *SubnetHandler) propagateDelete(ctx context.Context, from *ArubaSubnetBundle) error {
	if err := h.subnetRepository.Delete(ctx, from.Subnet); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

// Update re-applies the Subnet's tags. The CIDR, the VPC it sits in and its type are all fixed at
// creation on the Aruba side, so tags are the only part of a SECA subnet edit that can land.
func (h *SubnetHandler) Update(ctx context.Context, domain *subnetdom.Subnet) error {
	subnet, err := h.subnetConverter.FromSECAToAruba(domain)
	if err != nil {
		return err
	}

	return syncTags(ctx, h.subnetRepository, subnet, subnet.Spec.Tags, func(s *v1alpha1.Subnet) *[]string { return &s.Spec.Tags })
}
