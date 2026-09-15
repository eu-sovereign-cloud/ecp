package scheme_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"

	"github.com/eu-sovereign-cloud/ecp/resource/scheme"
)

// TestAddToScheme_CoversEveryCRD reads the generated CRDs rather than a hand-kept list, so a new
// slice whose CRD `make generate-api` emits fails here until it is added to AddToScheme.
func TestAddToScheme_CoversEveryCRD(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, scheme.AddToScheme(s))

	crds, err := filepath.Glob("../../charts/ecp/crds/*.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, crds, "no CRDs found; the test reads them relative to the repo root")

	for _, path := range crds {
		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var crd struct {
			Spec struct {
				Group string `json:"group"`
				Names struct {
					Kind     string `json:"kind"`
					ListKind string `json:"listKind"`
				} `json:"names"`
				Versions []struct {
					Name string `json:"name"`
				} `json:"versions"`
			} `json:"spec"`
		}
		require.NoError(t, yaml.Unmarshal(data, &crd), path)
		require.NotEmpty(t, crd.Spec.Versions, path)

		for _, v := range crd.Spec.Versions {
			gv := schema.GroupVersion{Group: crd.Spec.Group, Version: v.Name}
			for _, kind := range []string{crd.Spec.Names.Kind, crd.Spec.Names.ListKind} {
				assert.True(t, s.Recognizes(gv.WithKind(kind)), "%s: %s is not registered by AddToScheme", filepath.Base(path), gv.WithKind(kind))
			}
		}
	}
}
