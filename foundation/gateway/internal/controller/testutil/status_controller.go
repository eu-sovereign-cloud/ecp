// Package testutil provides test helpers for envtest-based integration tests.
package testutil

import (
	"context"
	"maps"
	"time"

	kerrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
)

// SimulateStatusController polls for a CR to appear and sets its status.state
// to "Ready", simulating what a real controller would do in a live cluster.
// Extra fields (e.g. "sizeGB") can be supplied via extraStatus and will be
// merged into the status object alongside "state" and "conditions".
func SimulateStatusController(ctx context.Context, dynClient dynamic.Interface, gvr schema.GroupVersionResource, namespace, name string, extraStatus map[string]interface{}) {
	ri := dynClient.Resource(gvr).Namespace(namespace)
	_ = wait.PollUntilContextTimeout(ctx, 50*time.Millisecond, 10*time.Second, true, func(ctx context.Context) (bool, error) {
		obj, err := ri.Get(ctx, name, metav1.GetOptions{})
		if kerrs.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}

		if err := unstructured.SetNestedField(obj.Object, buildStatus(extraStatus), "status"); err != nil {
			return false, err
		}

		_, err = ri.UpdateStatus(ctx, obj, metav1.UpdateOptions{})
		if kerrs.IsConflict(err) || kerrs.IsNotFound(err) {
			return false, nil
		}
		return err == nil, err
	})
}

func buildStatus(extra map[string]interface{}) map[string]interface{} {
	// state + conditions are required by every SECA CRD with a status subresource.
	status := map[string]interface{}{
		"state": "Ready",
		"conditions": []interface{}{
			map[string]interface{}{
				"state":            "Ready",
				"lastTransitionAt": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}
	maps.Copy(status, extra)
	return status
}
