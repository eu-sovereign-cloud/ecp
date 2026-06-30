package seca

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	authzport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/authz"
	rak8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role-assignment/backend/kubernetes"
	rolek8s "github.com/eu-sovereign-cloud/ecp/resource/authorization/v1/role/backend/kubernetes"
)

// discardLog returns a slog.Logger that discards all output (mirrors
// the helper in other test files in this package).
func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newTestScheme returns a runtime.Scheme with the authorization CRD types
// registered so the dynamic fake client can handle them.
func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = rolek8s.AddToScheme(s)
	_ = rak8s.AddToScheme(s)
	return s
}

// gvrToListKind maps each GVR the fake dynamic client needs to list to the
// corresponding List kind. This is required for resource names that contain
// hyphens (e.g. "role-assignments") because the fake client's default
// pluralization heuristic would produce "roleassignments" instead.
var gvrToListKind = map[schema.GroupVersionResource]string{
	rolek8s.RoleGVR:         "RoleList",
	rak8s.RoleAssignmentGVR: "RoleAssignmentList",
}

// newFakeClient creates a FakeDynamicClient with the correct GVR → list-kind
// mappings for authorization resources.
func newFakeClient(scheme *runtime.Scheme, objs ...runtime.Object) *fake.FakeDynamicClient {
	return fake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, objs...)
}

// TestCachedChecker_Start verifies that Start returns without error when the
// informer caches synchronise against an empty API server (empty fake client).
// The dynamicfake client returns empty list responses immediately, so cache
// sync should complete well within the context deadline.
func TestCachedChecker_Start(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	checker := NewCachedChecker(newFakeClient(scheme), discardLog())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := checker.Start(ctx); err != nil {
		t.Fatalf("Start returned unexpected error: %v", err)
	}
}

// TestCachedChecker_Authorize_EmptyCache verifies that Authorize returns
// kernel.ErrForbidden when the informer cache is empty (no roles or
// role assignments exist).
//
// Authorization correctness (the full scope + resource + verb matching logic)
// is already exercised exhaustively in TestEvaluate. This test focuses on the
// CachedChecker's adapter behaviour: reading from an empty cache → deny.
func TestCachedChecker_Authorize_EmptyCache(t *testing.T) {
	t.Parallel()

	scheme := newTestScheme()
	checker := NewCachedChecker(newFakeClient(scheme), discardLog())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := checker.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	claim := authzport.AuthorizationClaim{
		Provider:  "seca.compute",
		Resource:  "instances",
		Verb:      "list",
		Tenant:    "test-tenant",
		Region:    "us-west-1",
		Workspace: "ws1",
		Roles:     []string{"admin"},
	}

	err := checker.Authorize(context.Background(), claim)
	if err == nil {
		t.Fatal("expected error, got nil — empty cache should deny")
	}
	if !isErrForbidden(err) {
		t.Errorf("expected ErrForbidden, got %v", err)
	}
}

// isErrForbidden reports whether err is or wraps kernel.ErrForbidden.
func isErrForbidden(err error) bool {
	ke := kernel.AsError(err)
	return ke != nil && ke.Kind == kernel.KindForbidden
}
