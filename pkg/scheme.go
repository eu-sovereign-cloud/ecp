package pkg

import (
	k8upv1 "github.com/k8up-io/k8up/v2/api/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

// SetupScheme initializes a new runtime scheme and adds all known types to it.
// todo - unused as of now, but might be used in the future for custom resources
func SetupScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	AddToScheme(s)
	return s
}

func AddToScheme(s *runtime.Scheme) {
	_ = corev1.SchemeBuilder.AddToScheme(s)
	_ = k8upv1.SchemeBuilder.AddToScheme(s)
}
