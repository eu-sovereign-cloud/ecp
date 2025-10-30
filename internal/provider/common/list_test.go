package common

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"
)

var widgetGVR = schema.GroupVersionResource{Group: "example.io", Version: "v1", Resource: "widgets"}

// makeObj creates an unstructured object with optional namespace.
func makeObj(name string, ns string, labels map[string]string) *unstructured.Unstructured {
	labelMap := map[string]any{}
	for k, v := range labels {
		labelMap[k] = v
	}
	m := map[string]any{
		"apiVersion": "example.io/v1",
		"kind":       "Widget",
		"metadata": map[string]any{
			"name":   name,
			"labels": labelMap,
		},
	}
	if ns != "" {
		m["metadata"].(map[string]any)["namespace"] = ns
	}
	return &unstructured.Unstructured{Object: m}
}

func newFakeClient(objs ...runtime.Object) *fake.FakeDynamicClient {
	scheme := runtime.NewScheme() // empty scheme fine for unstructured
	return fake.NewSimpleDynamicClient(scheme, objs...)
}

func TestListResourcesSelector(t *testing.T) {
	client := newFakeClient(
		makeObj("w1", "", map[string]string{"env": "prod"}),
		makeObj("w2", "", map[string]string{"env": "dev"}),
	)
	ctx := context.Background()
	convert := func(u unstructured.Unstructured) (string, error) { return u.GetName(), nil }

	items, next, err := ListResources(ctx, client, widgetGVR, nil, convert, NewListOptions().Selector("env=prod"))
	if err != nil {
		t.Fatalf("ListResources returned error: %v", err)
	}
	if next != nil {
		t.Fatalf("expected no skip token, got %v", *next)
	}
	if len(items) != 1 || items[0] != "w1" {
		t.Fatalf("unexpected items: %#v", items)
	}
}

func TestGetResource(t *testing.T) {
	client := newFakeClient(makeObj("alpha", "", map[string]string{"tier": "gold"}))
	ctx := context.Background()
	convert := func(u unstructured.Unstructured) (string, error) { return u.GetName(), nil }
	item, err := GetResource(ctx, client, widgetGVR, "alpha", nil, convert, nil)
	if err != nil {
		t.Fatalf("GetResource returned error: %v", err)
	}
	if item != "alpha" {
		t.Fatalf("expected 'alpha', got %s", item)
	}
}

func TestListResourcesNilConvert(t *testing.T) {
	client := newFakeClient()
	ctx := context.Background()
	_, _, err := ListResources[string](ctx, client, widgetGVR, nil, nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil convert, got none")
	}
}

func TestGetResourceNilConvert(t *testing.T) {
	client := newFakeClient()
	ctx := context.Background()
	_, err := GetResource[string](ctx, client, widgetGVR, "anything", nil, nil, nil)
	if err == nil {
		t.Fatalf("expected error for nil convert, got none")
	}
}

func TestListResourcesNamespace(t *testing.T) {
	client := newFakeClient(
		makeObj("a", "ns1", map[string]string{"env": "prod"}),
		makeObj("b", "ns2", map[string]string{"env": "prod"}),
	)
	ctx := context.Background()
	convert := func(u unstructured.Unstructured) (string, error) { return u.GetName(), nil }
	items, _, err := ListResources(ctx, client, widgetGVR, nil, convert, NewListOptions().Namespace("ns1"))
	if err != nil {
		t.Fatalf("ListResources namespace returned error: %v", err)
	}
	if len(items) != 1 || items[0] != "a" {
		t.Fatalf("expected only 'a' from ns1, got %#v", items)
	}
}

func TestGetResourceNamespace(t *testing.T) {
	client := newFakeClient(makeObj("x", "ns-x", map[string]string{"env": "prod"}))
	ctx := context.Background()
	convert := func(u unstructured.Unstructured) (string, error) { return u.GetName(), nil }
	item, err := GetResource(ctx, client, widgetGVR, "x", nil, convert, NewGetOptions().Namespace("ns-x"))
	if err != nil {
		t.Fatalf("GetResource namespace returned error: %v", err)
	}
	if item != "x" {
		t.Fatalf("expected 'x', got %s", item)
	}
}
