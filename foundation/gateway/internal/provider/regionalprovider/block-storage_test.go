package regionalprovider

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	sdkstorage "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"
	"github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"

	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/block-storage/skus/v1"

	storage "github.com/eu-sovereign-cloud/ecp/foundation/api/block-storage"
	generatedv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/generated/types"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/internal/provider/kubernetes"
)

var cfg *rest.Config

// --- Helpers ---

// newStorageSKUCR constructs a typed StorageSKU CR.
func newStorageSKUCR(name, tenant string, labels map[string]string, iops, minVolumeSize int, skuType string, setVersionAndTimestamp bool) *skuv1.StorageSKU {
	if labels == nil {
		labels = map[string]string{}
	}
	cr := &skuv1.StorageSKU{
		TypeMeta:   metav1.TypeMeta{Kind: "StorageSKU", APIVersion: fmt.Sprintf("%s/%s", skuv1.StorageSKUGVR.Group, skuv1.StorageSKUGVR.Version)},
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels, Namespace: tenant},
		Spec:       generatedv1.StorageSkuSpec{Iops: iops, MinVolumeSize: minVolumeSize, Type: generatedv1.StorageSkuSpecType(skuType)},
	}
	if setVersionAndTimestamp {
		cr.SetCreationTimestamp(metav1.Time{Time: time.Unix(1700000000, 0)})
		cr.SetResourceVersion("1")
	}
	return cr
}

// toUnstructured converts a typed object to *unstructured.Unstructured (mirrors regions test helper).
func toUnstructured(t *testing.T, scheme *runtime.Scheme, obj runtime.Object) *unstructured.Unstructured {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	require.NoError(t, err)
	u := &unstructured.Unstructured{Object: m}
	gvk := obj.GetObjectKind().GroupVersionKind()
	if gvk.Empty() {
		gvks, _, err := scheme.ObjectKinds(obj)
		require.NoError(t, err)
		require.NotEmpty(t, gvks)
		gvk = gvks[0]
	}
	u.SetGroupVersionKind(gvk)
	return u
}

func extractSKUNames(sk []schema.StorageSku) []string {
	out := make([]string, len(sk))
	for i, s := range sk {
		out[i] = s.Metadata.Name
	}
	sort.Strings(out)
	return out
}

// --- Envtest lifecycle ---
func TestMain(m *testing.M) {
	wd, _ := os.Getwd()
	crdDir := filepath.Clean(filepath.Join(wd, "../../../../api/generated/crds/block-storage"))
	testEnvironment := &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		CRDDirectoryPaths:     []string{crdDir},
		DownloadBinaryAssets:  true,
		BinaryAssetsDirectory: filepath.Join(os.TempDir(), "envtest-binaries"),
	}
	var err error
	cfg, err = testEnvironment.Start()
	if err != nil {
		slog.Error("failed to start envtest", "error", err)
		os.Exit(1)
	}
	code := m.Run()
	if err := testEnvironment.Stop(); err != nil {
		slog.Error("failed to stop envtest", "error", err)
	}
	os.Exit(code)
}

// --- Tests ---

func TestStorageController_ListSKUs(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, storage.AddToScheme(scheme))

	dynClient, err := dynamic.NewForConfig(cfg)
	require.NoError(t, err)
	convert := func(u unstructured.Unstructured) (schema.StorageSku, error) {
		var crdStorageSKU skuv1.StorageSKU
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &crdStorageSKU); err != nil {
			return schema.StorageSku{}, err
		}
		return fromCRToSDKStorageSKU(crdStorageSKU), nil
	}
	storageSKUAdapter := kubernetes.NewAdapter(
		dynClient,
		skuv1.StorageSKUGVR,
		slog.Default(),
		convert,
	)
	sc := &StorageController{storageSKURepo: storageSKUAdapter}
	const (
		tenantA = "tenant-a"
		tenantB = "tenant-b"

		skuFast  = "fast"
		skuSlow  = "slow"
		skuThird = "third"
	)
	namespaceGVR := k8sschema.GroupVersionResource{Version: "v1", Resource: "namespaces"}

	// Create the namespace object
	namespaceObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": tenantA,
			},
		},
	}
	namespaceObj2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": tenantB,
			},
		},
	}
	ctx := context.Background()
	_, err = dynClient.Resource(namespaceGVR).Create(ctx, namespaceObj, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		require.NoError(t, err)
	}
	_, err = dynClient.Resource(namespaceGVR).Create(ctx, namespaceObj2, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		require.NoError(t, err)
	}
	nameFast := tenantA + "." + skuFast
	nameSlow := tenantA + "." + skuSlow
	nameThird := tenantA + "." + skuThird
	nameOtherTenant := tenantB + ".foreign"

	commonLabels := func(extra map[string]string) map[string]string {
		labels := map[string]string{tenantLabelKey: tenantA}
		for k, v := range extra {
			labels[k] = v
		}
		return labels
	}

	// Create CRs in the API server (no preset resourceVersion)
	for _, u := range []*unstructured.Unstructured{
		toUnstructured(t, scheme, newStorageSKUCR(nameFast, tenantA, commonLabels(map[string]string{"tier": "prod", "env": "prod"}), 5000, 10, string(generatedv1.StorageSkuTypeRemoteDurable), false)),
		toUnstructured(t, scheme, newStorageSKUCR(nameSlow, tenantA, commonLabels(map[string]string{"tier": "dev", "env": "staging"}), 1000, 20, string(generatedv1.StorageSkuTypeLocalDurable), false)),
		toUnstructured(t, scheme, newStorageSKUCR(nameThird, tenantA, commonLabels(map[string]string{"tier": "prod", "env": "staging", "rank": "3"}), 3000, 15, string(generatedv1.StorageSkuTypeRemoteDurable), false)),
		toUnstructured(t, scheme, newStorageSKUCR(nameOtherTenant, tenantB, map[string]string{tenantLabelKey: tenantB, "tier": "prod"}, 9000, 50, string(generatedv1.StorageSkuTypeRemoteDurable), false)),
	} {
		_, err := dynClient.Resource(skuv1.StorageSKUGVR).Namespace(u.GetNamespace()).Create(ctx, u, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	type tc struct {
		name      string
		selector  *string
		wantNames []string
	}

	complexServer := "tier=prod,env=prod"
	simple := "tier=prod"
	clientNumeric := "rank>2"
	none := "tier=qa"
	invalid := "tier=+=prod"
	onlyKey := "tier"
	setBased := "tier in (prod)"
	setBasedAndEq := "tier in (prod),env=staging"
	wildcardValue := "env=stag*"
	wildcardKeyVal := "t*r=pr*d"

	tests := []tc{
		{name: "all_no_selector", selector: nil, wantNames: []string{nameFast, nameSlow, nameThird}},
		{name: "simple_server_side_prefilter", selector: &simple, wantNames: []string{nameFast, nameThird}},
		{name: "complex_server_side_filter", selector: &complexServer, wantNames: []string{nameFast}},
		{name: "complex_client_numeric_filter", selector: &clientNumeric, wantNames: []string{nameThird}},
		{name: "no_matches", selector: &none, wantNames: []string{}},
		{name: "invalid_selector_skips_all", selector: &invalid, wantNames: []string{}},
		{name: "only_key_selector", selector: &onlyKey, wantNames: []string{nameFast, nameSlow, nameThird}},
		{name: "k8s_set_based_selector", selector: &setBased, wantNames: []string{nameFast, nameThird}},
		{name: "k8s_set_based_and_equality_selector", selector: &setBasedAndEq, wantNames: []string{nameThird}},
		{name: "wildcard_value", selector: &wildcardValue, wantNames: []string{nameSlow, nameThird}},
		{name: "wildcard_key_and_value", selector: &wildcardKeyVal, wantNames: []string{nameFast, nameThird}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			iter, err := sc.ListSKUs(ctx, tenantA, sdkstorage.ListSkusParams{Labels: tt.selector})
			require.NoError(t, err)
			require.ElementsMatch(t, tt.wantNames, extractSKUNames(iter.Items))
		})
	}
}

func TestStorageController_GetSKU(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, storage.AddToScheme(scheme))

	dynClient, err := dynamic.NewForConfig(cfg)
	require.NoError(t, err)
	convert := func(u unstructured.Unstructured) (schema.StorageSku, error) {
		var crdStorageSKU skuv1.StorageSKU
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &crdStorageSKU); err != nil {
			return schema.StorageSku{}, err
		}
		return fromCRToSDKStorageSKU(crdStorageSKU), nil
	}
	storageSKUAdapter := kubernetes.NewAdapter(
		dynClient,
		skuv1.StorageSKUGVR,
		slog.Default(),
		convert,
	)
	sc := &StorageController{storageSKURepo: storageSKUAdapter, logger: slog.Default()}

	const tenant = "tenant-a"
	const skuID = "only"
	namespaceGVR := k8sschema.GroupVersionResource{Version: "v1", Resource: "namespaces"}

	// Create the namespace object
	namespaceObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": tenant,
			},
		},
	}

	ctx := context.Background()
	_, err = dynClient.Resource(namespaceGVR).Create(ctx, namespaceObj, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		require.NoError(t, err)
	}
	u := toUnstructured(t, scheme, newStorageSKUCR(skuID, tenant, map[string]string{tenantLabelKey: tenant, "tier": "prod"}, 7500, 10, string(generatedv1.StorageSkuTypeRemoteDurable), false))

	_, err = dynClient.Resource(skuv1.StorageSKUGVR).Namespace(u.GetNamespace()).Create(ctx, u, metav1.CreateOptions{})
	if err != nil && !k8serrors.IsAlreadyExists(err) { // ignore if previously created by another test
		require.NoError(t, err)
	}

	t.Run("get_existing", func(t *testing.T) {
		sku, err := sc.GetSKU(ctx, tenant, skuID)
		require.NoError(t, err)
		require.NotNil(t, sku)
		require.Equal(t, skuID, sku.Metadata.Name)
		require.NotNil(t, sku.Spec)
		require.Equal(t, 7500, sku.Spec.Iops)
	})

	t.Run("get_nonexistent", func(t *testing.T) {
		_, err := sc.GetSKU(ctx, tenant, "missing")
		require.Error(t, err)
	})
}
