package plugin

import (
	"fmt"

	v1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	corev1 "k8s.io/api/core/v1"
)

type conditioned interface {
	GetCondition(v1.ConditionType) v1.Condition
}

// checkReady returns (true, nil) when the resource is ready, (false, err) when
// the provider has reported a reconcile error, or (false, nil) while provisioning.
func checkReady(r conditioned, kind string) (bool, error) {
	if r.GetCondition(v1.TypeReady).Status == corev1.ConditionTrue {
		return true, nil
	}
	if synced := r.GetCondition(v1.TypeSynced); synced.Status == corev1.ConditionFalse &&
		synced.Reason == v1.ReasonReconcileError {
		return false, fmt.Errorf("provider failed to reconcile %s: %s", kind, synced.Message)
	}
	return false, nil
}
