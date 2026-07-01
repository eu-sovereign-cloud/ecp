package seca

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/gateway/internal/metrics"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment"
	rak8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment/backend/kubernetes"
	rolek8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role/backend/kubernetes"
)

// defaultResync is the period after which the informer re-lists all objects to
// detect changes that were missed by the watch stream.
const defaultResync = 5 * time.Minute

// CachedChecker is the informer-backed SECA RBAC implementation of authzport.Checker.
//
// It is identical in policy semantics to Checker but avoids Kubernetes API-server
// round-trips on every Authorize call by reading from an in-process informer cache.
// The cache is kept current by watch events for RoleGVR and RoleAssignmentGVR.
//
// Lifecycle: call Start once at server startup (after the dynamic client is ready)
// and pass the server's context so the informers are stopped on shutdown.
//
// This is requirement 2.3 from the design. For the un-cached variant see Checker.
type CachedChecker struct {
	factory dynamicinformer.DynamicSharedInformerFactory
	log     *slog.Logger
}

// NewCachedChecker creates a CachedChecker backed by the given Kubernetes dynamic client.
// Call Start before any Authorize call to warm up the cache.
func NewCachedChecker(dynClient dynamic.Interface, log *slog.Logger) *CachedChecker {
	return &CachedChecker{
		factory: dynamicinformer.NewDynamicSharedInformerFactory(dynClient, defaultResync),
		log:     log,
	}
}

// Start pre-registers informers for Roles and RoleAssignments, starts them, and
// blocks until both caches are synced. Returns an error if the context is cancelled
// before sync completes (which means the API server is unreachable at startup).
func (c *CachedChecker) Start(ctx context.Context) error {
	// Calling ForResource before Start ensures both informers are registered with the
	// factory; the factory starts only the informers that have been requested.
	_ = c.factory.ForResource(rolek8s.RoleGVR)
	_ = c.factory.ForResource(rak8s.RoleAssignmentGVR)

	c.factory.Start(ctx.Done())

	synced := c.factory.WaitForCacheSync(ctx.Done())
	for gvr, ok := range synced {
		if !ok {
			return fmt.Errorf("informer cache sync timed out for %s", gvr.Resource)
		}
	}
	return nil
}

// Authorize implements authzport.Checker.
// Returns DecisionAllowed with nil error when the claim is permitted.
// Returns DecisionDenied with kernel.ErrForbidden when policy denies the operation.
// Returns DecisionError with a kernel.KindInternal error when the informer cache
// cannot be read, ensuring a technical failure is never silently disguised as an
// authorization denial.
func (c *CachedChecker) Authorize(ctx context.Context, claim authzport.AuthorizationClaim) (authzport.Decision, error) {
	fetchStart := time.Now()
	rolesByName, assignments, err := c.loadFromCache(claim.Tenant)
	metrics.ObserveRBACFetch("cached", time.Since(fetchStart))
	if err != nil {
		c.log.ErrorContext(ctx, "seca rbac (cached): failed to load from informer cache", slog.Any("error", err))
		return authzport.DecisionError, kernel.NewError(kernel.KindInternal, fmt.Errorf("load policy data from cache: %w", err))
	}

	if Evaluate(claim, rolesByName, assignments) {
		return authzport.DecisionAllowed, nil
	}
	return authzport.DecisionDenied, kernel.ErrForbidden
}

// loadFromCache reads Roles and RoleAssignments from the informer cache for the
// given tenant namespace.
func (c *CachedChecker) loadFromCache(tenant string) (map[string]*roledom.Role, []*radom.RoleAssignment, error) {
	ns := k8sadapter.ComputeNamespace(&resource.Scope{Tenant: tenant})

	rawRoles, err := c.factory.ForResource(rolek8s.RoleGVR).Lister().ByNamespace(ns).List(labels.Everything())
	if err != nil {
		return nil, nil, fmt.Errorf("list roles from cache (ns=%s): %w", ns, err)
	}

	rawAssignments, err := c.factory.ForResource(rak8s.RoleAssignmentGVR).Lister().ByNamespace(ns).List(labels.Everything())
	if err != nil {
		return nil, nil, fmt.Errorf("list assignments from cache (ns=%s): %w", ns, err)
	}

	rolesByName := make(map[string]*roledom.Role, len(rawRoles))
	for _, obj := range rawRoles {
		u, err := toUnstructured(obj)
		if err != nil {
			c.log.Warn("seca rbac (cached): skip malformed role in cache", slog.Any("error", err))
			continue
		}
		r, err := rolek8s.RoleFromCR(u)
		if err != nil {
			c.log.Warn("seca rbac (cached): skip unconvertible role", slog.Any("error", err))
			continue
		}
		rolesByName[r.GetName()] = r
	}

	assignments := make([]*radom.RoleAssignment, 0, len(rawAssignments))
	for _, obj := range rawAssignments {
		u, err := toUnstructured(obj)
		if err != nil {
			c.log.Warn("seca rbac (cached): skip malformed assignment in cache", slog.Any("error", err))
			continue
		}
		ra, err := rak8s.RoleAssignmentFromCR(u)
		if err != nil {
			c.log.Warn("seca rbac (cached): skip unconvertible assignment", slog.Any("error", err))
			continue
		}
		assignments = append(assignments, ra)
	}

	return rolesByName, assignments, nil
}

// toUnstructured extracts the *unstructured.Unstructured from a runtime.Object returned
// by an informer cache listing. Informers always store unstructured objects when the
// dynamic client is used.
func toUnstructured(obj runtime.Object) (*unstructured.Unstructured, error) {
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("unexpected object type in informer cache: %T", obj)
	}
	return u, nil
}
