//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	resourcescheme "github.com/eu-sovereign-cloud/ecp/resource/scheme"
)

// TestSchemeKindsAreServed asserts every CR kind a delegator registers through resource/scheme
// is served by the cluster under the same group, version and kind. The delegator's typed client
// resolves each object through that mapping, so a kind registered under a group or version no
// installed CRD serves would otherwise only fail when its controller first touches it.
func TestSchemeKindsAreServed(t *testing.T) {
	s := runtime.NewScheme()
	require.NoError(t, resourcescheme.AddToScheme(s))

	checked := 0
	for gvk, typ := range s.AllKnownTypes() {
		// Every group version also carries the metav1 option types, and each CR its List kind;
		// neither is a served resource.
		if !strings.HasPrefix(typ.PkgPath(), "github.com/eu-sovereign-cloud/ecp/resource/") || strings.HasSuffix(gvk.Kind, "List") {
			continue
		}
		checked++
		t.Run(gvk.String(), func(t *testing.T) {
			_, err := k8sClient.RESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
			require.NoError(t, err)
		})
	}
	require.NotZero(t, checked, "resource/scheme registered no SECA kinds")
}
