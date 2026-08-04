package handler

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	backend "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	res "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	igwdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	netk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/delegated"
	mutator_bypass "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/mutator"
	resolver_bypass "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/resolver"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/converter"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/repository"
)

// Ensure NetworkHandler implements the Network interface
var _ netk8s.NetworkPlugin = (*NetworkHandler)(nil)

// NetworkHandler handles Network resources by interacting with Aruba VPC.
// It is responsible for translating Network resources to Aruba VPC
// and managing their lifecycle (Create/Delete).
type NetworkHandler struct {
	wsRepository    persistence.ReaderRepo[*wsdom.Workspace]
	igwRepository   persistence.ReaderRepo[*igwdom.InternetGateway]
	vpcRepository   repository.Repository[*v1alpha1.VPC, *v1alpha1.VPCList]
	prjRepository   repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList]
	netConverter    converter.Converter[*netdom.Network, *v1alpha1.VPC]
	wsConverter     converter.Converter[*wsdom.Workspace, *v1alpha1.Project]
	createDelegated *delegated.GenericDelegated[*netdom.Network, *SecaNetworkBundle, *ArubaNetworkBundle]
	deleteDelegated *delegated.GenericDelegated[*netdom.Network, *SecaNetworkBundle, *ArubaNetworkBundle]
}

type SecaNetworkBundle struct {
	Network   *netdom.Network
	Workspace *wsdom.Workspace
}

type ArubaNetworkBundle struct {
	VPC     *v1alpha1.VPC
	Project *v1alpha1.Project
}

// NewNetworkHandler creates a new NetworkHandler with the provided repositories and converters.
// It sets up the necessary delegated operations for creating and deleting Network resources.
// The handler uses bypass mutators since no mutation is needed on the Aruba VPC objects.
func NewNetworkHandler(
	wsRepo persistence.ReaderRepo[*wsdom.Workspace],
	igwRepo persistence.ReaderRepo[*igwdom.InternetGateway],
	vpcRepo repository.Repository[*v1alpha1.VPC, *v1alpha1.VPCList],
	prjRepo repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList],
	netConv converter.Converter[*netdom.Network, *v1alpha1.VPC],
	wsConv converter.Converter[*wsdom.Workspace, *v1alpha1.Project],
) *NetworkHandler {
	handler := &NetworkHandler{
		wsRepository:  wsRepo,
		igwRepository: igwRepo,
		vpcRepository: vpcRepo,
		prjRepository: prjRepo,
		netConverter:  netConv,
		wsConverter:   wsConv,
	}

	handler.createDelegated = delegated.NewDelegated(
		handler.resolveSecaNetworkDependencies,
		handler.FromSECABundleToAruba,
		handler.resolveArubaNetworkDependencies,
		mutator_bypass.BypassMutateFunc[*ArubaNetworkBundle, *SecaNetworkBundle],
		handler.propagateCreate,
		handler.checkVpcCreated,
	)

	handler.deleteDelegated = delegated.NewDelegated(
		handler.BypassDependencyResolver,
		handler.FromSECABundleToAruba,
		resolver_bypass.BypassResolveDependenciesFunc[*ArubaNetworkBundle],
		mutator_bypass.BypassMutateFunc[*ArubaNetworkBundle, *SecaNetworkBundle],
		handler.propagateDelete,
		handler.checkVpcDeleted,
	)

	return handler
}

// Create creates a new Network by creating an Aruba VPC.
func (h *NetworkHandler) Create(ctx context.Context, domain *netdom.Network) error {
	return h.createDelegated.Do(ctx, domain)
}

// Delete deletes an existing Network by deleting the corresponding Aruba VPC.
func (h *NetworkHandler) Delete(ctx context.Context, domain *netdom.Network) error {
	return h.deleteDelegated.Do(ctx, domain)
}

// checkVpcCreated reports whether the Aruba VPC already exists and has reached
// the active phase.
func (h *NetworkHandler) checkVpcCreated(ctx context.Context, _ *SecaNetworkBundle, bundle *ArubaNetworkBundle) (bool, error) {
	observed := bundle.VPC.DeepCopy()

	if err := h.vpcRepository.Load(ctx, observed); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil // Not created yet, it must be created.
		}

		return false, err
	}

	return observed.Status.Phase == v1alpha1.ResourcePhaseActive, nil
}

// checkVpcDeleted reports whether the Aruba VPC is gone.
func (h *NetworkHandler) checkVpcDeleted(ctx context.Context, _ *SecaNetworkBundle, bundle *ArubaNetworkBundle) (bool, error) {
	observed := bundle.VPC.DeepCopy()

	if err := h.vpcRepository.Load(ctx, observed); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil // Gone, deletion is complete.
		}

		return false, err
	}

	return false, nil // Still present, deletion is in progress.
}

func (h *NetworkHandler) BypassDependencyResolver(_ context.Context, domain *netdom.Network) (*SecaNetworkBundle, error) {
	return &SecaNetworkBundle{
		Network: domain,
	}, nil
}

// resolveSecaNetworkDependencies gates the VPC creation on the workspace being active and on an
// InternetGateway existing in that workspace.
//
// Only the InternetGateway gates: an Aruba VPC always provides internet egress, so the SECA
// model must carry the gateway that represents it before the VPC exists. No RouteTable is
// required - the operator creates VPCs with preset=false, so Aruba never derives a subnet or a
// route table from the VPC, and both are created independently afterwards.
func (h *NetworkHandler) resolveSecaNetworkDependencies(ctx context.Context, domain *netdom.Network) (*SecaNetworkBundle, error) {
	ws, err := loadActiveWorkspace(ctx, h.wsRepository, domain)
	if err != nil {
		return nil, err
	}

	var igws []*igwdom.InternetGateway
	if _, err := h.igwRepository.List(ctx, res.ListParams{
		Scope: res.Scope{
			Tenant:    domain.GetTenant(),
			Workspace: domain.GetWorkspace(),
		},
	}, &igws); err != nil {
		return nil, err
	}

	if len(igws) == 0 {
		return nil, backend.ErrStillProcessing // No internet gateway yet, wait for one to be created
	}

	return &SecaNetworkBundle{
		Network:   domain,
		Workspace: ws,
	}, nil
}

func (h *NetworkHandler) resolveArubaNetworkDependencies(ctx context.Context, arubaBundle *ArubaNetworkBundle) (*ArubaNetworkBundle, error) {
	if err := loadActiveProject(ctx, h.prjRepository, arubaBundle.Project); err != nil {
		return nil, err
	}

	return arubaBundle, nil
}

func (h *NetworkHandler) FromSECABundleToAruba(from *SecaNetworkBundle) (*ArubaNetworkBundle, error) {
	response := &ArubaNetworkBundle{}

	if from.Workspace != nil {
		prj, err := h.wsConverter.FromSECAToAruba(from.Workspace)
		if err != nil {
			return nil, err
		}

		response.Project = prj
	}

	vpc, err := h.netConverter.FromSECAToAruba(from.Network)
	if err != nil {
		return nil, err
	}

	response.VPC = vpc

	return response, nil
}

// propagateCreate creates the Aruba VPC. It is idempotent: because the create is
// (re)issued on every pass until the resource becomes active, an already
// existing resource is not treated as an error.
func (h *NetworkHandler) propagateCreate(ctx context.Context, from *ArubaNetworkBundle) error {
	if err := h.vpcRepository.Create(ctx, from.VPC); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// propagateDelete deletes the Aruba VPC. It is idempotent: because the delete is
// (re)issued on every pass until the resource is gone, an already missing
// resource is not treated as an error.
func (h *NetworkHandler) propagateDelete(ctx context.Context, from *ArubaNetworkBundle) error {
	if err := h.vpcRepository.Delete(ctx, from.VPC); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

// Update re-applies the VPC's tags, which is the whole of what an Aruba VPC lets an update change
// (see update.go). It carries a SECA label edit through to the provider.
func (h *NetworkHandler) Update(ctx context.Context, domain *netdom.Network) error {
	vpc, err := h.netConverter.FromSECAToAruba(domain)
	if err != nil {
		return err
	}

	return syncTags(ctx, h.vpcRepository, vpc, vpc.Spec.Tags, func(v *v1alpha1.VPC) *[]string { return &v.Spec.Tags })
}
