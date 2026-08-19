package metrics

import (
	"context"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	k8sadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
)

func TestRegisterUpstreamObserver_RecordsHistogram(t *testing.T) {
	RegisterUpstreamObserver()
	t.Cleanup(func() { k8sadapter.SetUpstreamObserver(nil) })

	before := upstreamSampleCount(t, "namespaces", "core", "create", "ok")

	cs := k8sfake.NewClientset()
	if _, err := k8sadapter.CreateNamespace(context.Background(), cs, "ns-observe", nil); err != nil {
		t.Fatalf("CreateNamespace: %v", err)
	}

	after := upstreamSampleCount(t, "namespaces", "core", "create", "ok")
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
