package providers

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ProviderInterface todo - this is a temporary interface, we need to replace the type of req with a more specific type
type ProviderInterface interface {
	ValidateRequest(ctx context.Context, req *interface{}) error
	GenerateXRD(ctx context.Context, req *interface{}) (*unstructured.Unstructured, error)
	GetCompositionRef(providerType string) string
}
