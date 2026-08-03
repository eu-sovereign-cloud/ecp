package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
)

func TestRegisterUpstreamObserver_RecordsHistogram(t *testing.T) {
	RegisterUpstreamObserver()
	t.Cleanup(func() { k8sadapter.SetUpstreamObserver(nil) })

	before := upstreamSampleCount(t, "workspaces", "workspace.v1.secapi.cloud", "create", "ok")

	k8sadapter.SetUpstreamObserver(promUpstreamObserver{})
	promUpstreamObserver{}.Observe("workspaces", "workspace.v1.secapi.cloud", "create", "ok", 5*time.Millisecond)

	after := upstreamSampleCount(t, "workspaces", "workspace.v1.secapi.cloud", "create", "ok")
	if after < before+1 {
		t.Fatalf("expected sample count to increase, before=%v after=%v", before, after)
	}
}

func upstreamSampleCount(t *testing.T, resource, group, operation, result string) float64 {
	t.Helper()
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range mfs {
		if mf.GetName() != "ecp_gateway_upstream_kube_request_duration_seconds" {
			continue
		}
		for _, m := range mf.GetMetric() {
			labels := map[string]string{}
			for _, l := range m.GetLabel() {
				labels[l.GetName()] = l.GetValue()
			}
			if labels["resource"] == resource &&
				labels["group"] == group &&
				labels["operation"] == operation &&
				labels["result"] == result {
				if h := m.GetHistogram(); h != nil {
					return float64(h.GetSampleCount())
				}
			}
		}
	}
	return 0
}
