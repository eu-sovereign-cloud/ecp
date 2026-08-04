package handler

import (
	"context"

	"github.com/Arubacloud/arubacloud-resource-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	persistence "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
	pipk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip/backend/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"

	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/delegated"
	mutator_bypass "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/mutator"
	resolver_bypass "github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/adapter/generic/resolver"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/converter"
	"github.com/eu-sovereign-cloud/ecp/csp/aruba/pkg/port/repository"
)

// Ensure PublicIpHandler implements the PublicIp interface
var _ pipk8s.PublicIpPlugin = (*PublicIpHandler)(nil)

// PublicIpHandler handles PublicIp resources by interacting with Aruba ElasticIP.
// It is responsible for translating PublicIp resources to Aruba ElasticIP
// and managing their lifecycle (Create/Delete).
type PublicIpHandler struct {
	wsRepository    persistence.ReaderRepo[*wsdom.Workspace]
	eipRepository   repository.Repository[*v1alpha1.ElasticIP, *v1alpha1.ElasticIPList]
	prjRepository   repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList]
	pipConverter    converter.Converter[*publicipdom.PublicIp, *v1alpha1.ElasticIP]
	wsConverter     converter.Converter[*wsdom.Workspace, *v1alpha1.Project]
	createDelegated *delegated.GenericDelegated[*publicipdom.PublicIp, *SecaPublicIpBundle, *ArubaPublicIpBundle]
	deleteDelegated *delegated.GenericDelegated[*publicipdom.PublicIp, *SecaPublicIpBundle, *ArubaPublicIpBundle]
}

type SecaPublicIpBundle struct {
	PublicIp  *publicipdom.PublicIp
	Workspace *wsdom.Workspace
}

type ArubaPublicIpBundle struct {
	ElasticIP *v1alpha1.ElasticIP
	Project   *v1alpha1.Project
}

// NewPublicIpHandler creates a new PublicIpHandler with the provided repositories and converters.
// It sets up the necessary delegated operations for creating and deleting PublicIp resources.
// The handler uses bypass mutators since no mutation is needed on the Aruba ElasticIP objects.
func NewPublicIpHandler(
	wsRepo persistence.ReaderRepo[*wsdom.Workspace],
	eipRepo repository.Repository[*v1alpha1.ElasticIP, *v1alpha1.ElasticIPList],
	prjRepo repository.Repository[*v1alpha1.Project, *v1alpha1.ProjectList],
	pipConv converter.Converter[*publicipdom.PublicIp, *v1alpha1.ElasticIP],
	wsConv converter.Converter[*wsdom.Workspace, *v1alpha1.Project],
) *PublicIpHandler {
	handler := &PublicIpHandler{
		wsRepository:  wsRepo,
		eipRepository: eipRepo,
		prjRepository: prjRepo,
		pipConverter:  pipConv,
		wsConverter:   wsConv,
	}

	handler.createDelegated = delegated.NewDelegated(
		handler.resolveSecaPublicIpDependencies,
		handler.FromSECABundleToAruba,
		handler.resolveArubaPublicIpDependencies,
		mutator_bypass.BypassMutateFunc[*ArubaPublicIpBundle, *SecaPublicIpBundle],
		handler.propagateCreate,
		handler.checkEipCreated,
	)

	handler.deleteDelegated = delegated.NewDelegated(
		handler.BypassDependencyResolver,
		handler.FromSECABundleToAruba,
		resolver_bypass.BypassResolveDependenciesFunc[*ArubaPublicIpBundle],
		mutator_bypass.BypassMutateFunc[*ArubaPublicIpBundle, *SecaPublicIpBundle],
		handler.propagateDelete,
		handler.checkEipDeleted,
	)

	return handler
}

// Create creates a new PublicIp by creating an Aruba ElasticIP.
func (h *PublicIpHandler) Create(ctx context.Context, domain *publicipdom.PublicIp) error {
	return h.createDelegated.Do(ctx, domain)
}

// Delete deletes an existing PublicIp by deleting the corresponding Aruba ElasticIP.
func (h *PublicIpHandler) Delete(ctx context.Context, domain *publicipdom.PublicIp) error {
	return h.deleteDelegated.Do(ctx, domain)
}

// checkEipCreated reports whether the Aruba ElasticIP already exists and has
// reached the active phase.
func (h *PublicIpHandler) checkEipCreated(ctx context.Context, _ *SecaPublicIpBundle, bundle *ArubaPublicIpBundle) (bool, error) {
	observed := bundle.ElasticIP.DeepCopy()

	if err := h.eipRepository.Load(ctx, observed); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil // Not created yet, it must be created.
		}

		return false, err
	}

	return observed.Status.Phase == v1alpha1.ResourcePhaseActive, nil
}

// checkEipDeleted reports whether the Aruba ElasticIP is gone.
func (h *PublicIpHandler) checkEipDeleted(ctx context.Context, _ *SecaPublicIpBundle, bundle *ArubaPublicIpBundle) (bool, error) {
	observed := bundle.ElasticIP.DeepCopy()

	if err := h.eipRepository.Load(ctx, observed); err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil // Gone, deletion is complete.
		}

		return false, err
	}

	return false, nil // Still present, deletion is in progress.
}

func (h *PublicIpHandler) BypassDependencyResolver(_ context.Context, domain *publicipdom.PublicIp) (*SecaPublicIpBundle, error) {
	return &SecaPublicIpBundle{
		PublicIp: domain,
	}, nil
}

func (h *PublicIpHandler) resolveSecaPublicIpDependencies(ctx context.Context, domain *publicipdom.PublicIp) (*SecaPublicIpBundle, error) {
	ws, err := loadActiveWorkspace(ctx, h.wsRepository, domain)
	if err != nil {
		return nil, err
	}

	return &SecaPublicIpBundle{
		PublicIp:  domain,
		Workspace: ws,
	}, nil
}

func (h *PublicIpHandler) resolveArubaPublicIpDependencies(ctx context.Context, arubaBundle *ArubaPublicIpBundle) (*ArubaPublicIpBundle, error) {
	if err := loadActiveProject(ctx, h.prjRepository, arubaBundle.Project); err != nil {
		return nil, err
	}

	return arubaBundle, nil
}

func (h *PublicIpHandler) FromSECABundleToAruba(from *SecaPublicIpBundle) (*ArubaPublicIpBundle, error) {
	response := &ArubaPublicIpBundle{}

	if from.Workspace != nil {
		prj, err := h.wsConverter.FromSECAToAruba(from.Workspace)
		if err != nil {
			return nil, err
		}

		response.Project = prj
	}

	eip, err := h.pipConverter.FromSECAToAruba(from.PublicIp)
	if err != nil {
		return nil, err
	}

	response.ElasticIP = eip

	return response, nil
}

// propagateCreate creates the Aruba ElasticIP. It is idempotent: because the
// create is (re)issued on every pass until the resource becomes active, an
// already existing resource is not treated as an error.
func (h *PublicIpHandler) propagateCreate(ctx context.Context, from *ArubaPublicIpBundle) error {
	if err := h.eipRepository.Create(ctx, from.ElasticIP); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

// propagateDelete deletes the Aruba ElasticIP. It is idempotent: because the
// delete is (re)issued on every pass until the resource is gone, an already
// missing resource is not treated as an error.
func (h *PublicIpHandler) propagateDelete(ctx context.Context, from *ArubaPublicIpBundle) error {
	if err := h.eipRepository.Delete(ctx, from.ElasticIP); err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	return nil
}

// Update re-applies the ElasticIP's tags. The address itself is allocated by Aruba and cannot be
// changed, which the converter already rejects on the create path.
func (h *PublicIpHandler) Update(ctx context.Context, domain *publicipdom.PublicIp) error {
	eip, err := h.pipConverter.FromSECAToAruba(domain)
	if err != nil {
		return err
	}

	return syncTags(ctx, h.eipRepository, eip, eip.Spec.Tags, func(e *v1alpha1.ElasticIP) *[]string { return &e.Spec.Tags })
}
