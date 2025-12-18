package kubernetes

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type TestResource struct {
	Name      string
	Namespace string
	Value     string
	UID       string
}

func (r *TestResource) SetName(name string) {
	// TODO implement me
	panic("implement me")
}

func (r *TestResource) SetNamespace(namespace string) {
	// TODO implement me
	panic("implement me")
}

func (r *TestResource) GetName() string {
	return r.Name
}

func (r *TestResource) GetNamespace() string {
	return r.Namespace
}

func toK8s(r *TestResource) (*unstructured.Unstructured, error) {
	u := &unstructured.Unstructured{}

	u.SetUnstructuredContent(map[string]interface{}{
		"spec": map[string]interface{}{
			"value": r.Value,
		},
	})
	u.SetGroupVersionKind(schema.GroupVersionKind{Group: "test", Version: "v1", Kind: "Test"})
	u.SetName(r.Name)
	u.SetNamespace(r.Namespace)
	u.SetUID(types.UID(r.UID))
	return u, nil
}

func toDomain(obj client.Object) (*TestResource, error) {
	u := obj.(*unstructured.Unstructured)
	val, _, _ := unstructured.NestedString(u.Object, "spec", "value")
	return &TestResource{
		Name:      u.GetName(),
		Namespace: u.GetNamespace(),
		Value:     val,
		UID:       string(u.GetUID()),
	}, nil
}

func TestAdapter_Create_Update(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "test", Version: "v1", Resource: "tests"}
	scheme := runtime.NewScheme()
	client := fake.NewSimpleDynamicClient(scheme)

	adapter := &Adapter[*TestResource]{
		client:   client,
		gvr:      gvr,
		logger:   slog.Default(),
		toDomain: toDomain,
		toK8s:    toK8s,
	}

	ctx := context.Background()

	// Test Create
	res := &TestResource{
		Name:      "test-1",
		Namespace: "default",
		Value:     "initial",
		UID:       "myuid",
	}

	err := adapter.Create(ctx, res)
	require.NoError(t, err)
	// fake client sets UID
	require.NotEmpty(t, res.UID, "UID should be set after create")
	require.Equal(t, "initial", res.Value)

	// Test Update
	res.Value = "updated"
	err = adapter.Update(ctx, res)
	require.NoError(t, err)
	require.Equal(t, "updated", res.Value)

	// Verify in client
	u, err := client.Resource(gvr).Namespace("default").Get(ctx, "test-1", metav1.GetOptions{})
	require.NoError(t, err)
	val, _, _ := unstructured.NestedString(u.Object, "spec", "value")
	require.Equal(t, "updated", val)
}
