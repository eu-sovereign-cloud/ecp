package common

import (
	"context"
	"fmt"
	"log/slog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/eu-sovereign-cloud/ecp/internal/validation/filter"
)

// DefaultUnstructuredConverter returns a conversion function that decodes the unstructured object into T using the runtime default converter.
func DefaultUnstructuredConverter[T any]() func(unstructured.Unstructured) (T, error) {
	return func(obj unstructured.Unstructured) (T, error) {
		var out T
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, &out); err != nil {
			var zero T
			return zero, fmt.Errorf("default converter failed: %w", err)
		}
		return out, nil
	}
}

// Adapter composes the default unstructured conversion for a CRD type R with a mapping function to
// produce a target type S. It returns a convert closure suitable for ListResources / GetResource.
func Adapter[R any, S any](mapFn func(R) (S, error)) func(unstructured.Unstructured) (S, error) {
	base := DefaultUnstructuredConverter[R]()
	return func(u unstructured.Unstructured) (S, error) {
		cr, err := base(u)
		if err != nil {
			var zero S
			return zero, err
		}
		return mapFn(cr)
	}
}

// ListOptions provides a builder-style configuration for ListResources (now without logger).
type ListOptions struct {
	namespace *string
	limit     int
	skipToken *string
	selector  *string
}

func NewListOptions() *ListOptions                      { return &ListOptions{} }
func (o *ListOptions) Namespace(ns string) *ListOptions { o.namespace = &ns; return o }
func (o *ListOptions) Limit(limit int) *ListOptions {
	if limit > 0 {
		o.limit = limit
	}
	return o
}

func (o *ListOptions) SkipToken(token string) *ListOptions {
	if token != "" {
		o.skipToken = &token
	}
	return o
}

func (o *ListOptions) Selector(sel string) *ListOptions {
	if sel != "" {
		o.selector = &sel
	}
	return o
}

// GetOptions provides a builder-style configuration for GetResource (now without logger).
type GetOptions struct {
	namespace *string
}

func NewGetOptions() *GetOptions                      { return &GetOptions{} }
func (o *GetOptions) Namespace(ns string) *GetOptions { o.namespace = &ns; return o }

// ListResources lists resources using builder options. Logger is passed explicitly.
func ListResources[T any](
	ctx context.Context,
	client dynamic.Interface,
	gvr schema.GroupVersionResource,
	logger *slog.Logger,
	convert func(unstructured.Unstructured) (T, error),
	opts *ListOptions,
) ([]T, *string, error) {
	if convert == nil {
		return nil, nil, fmt.Errorf("ListResources: convert function must be provided")
	}
	if opts == nil {
		opts = NewListOptions()
	}
	lo := metav1.ListOptions{}
	if opts.limit > 0 {
		lo.Limit = int64(opts.limit)
	}
	if opts.skipToken != nil {
		lo.Continue = *opts.skipToken
	}
	selectorStr := ""
	if opts.selector != nil {
		selectorStr = *opts.selector
		lo.LabelSelector = filter.K8sSelectorForAPI(selectorStr)
	}
	var ri dynamic.ResourceInterface
	ri = client.Resource(gvr)
	if opts.namespace != nil {
		ri = client.Resource(gvr).Namespace(*opts.namespace)
	}
	ulist, err := ri.List(ctx, lo)
	if err != nil {
		if logger != nil {
			logger.ErrorContext(ctx, "failed to list resources", slog.String("resource", gvr.Resource), slog.Any("error", err))
		}
		return nil, nil, fmt.Errorf("failed to list resources for %s: %w", gvr.Resource, err)
	}
	result := make([]T, 0, len(ulist.Items))
	for _, item := range ulist.Items {
		if selectorStr != "" {
			match, k8sHandled, e := filter.MatchLabels(item.GetLabels(), selectorStr)
			if e != nil {
				if logger != nil {
					logger.WarnContext(ctx, "invalid selector, skipping", slog.String("resource", gvr.Resource), slog.String("selector", selectorStr), slog.Any("error", e))
				}
				continue
			}
			if !match && !k8sHandled {
				continue
			}
		}
		converted, err := convert(item)
		if err != nil {
			if logger != nil {
				logger.ErrorContext(ctx, "conversion failed", slog.String("resource", gvr.Resource), slog.Any("error", err))
			}
			return nil, nil, fmt.Errorf("failed to convert %s: %w", gvr.Resource, err)
		}
		result = append(result, converted)
	}
	next := ulist.GetContinue()
	if next == "" {
		return result, nil, nil
	}
	return result, &next, nil
}

// GetResource fetches and converts a single resource. Logger is passed explicitly.
func GetResource[T any](
	ctx context.Context,
	client dynamic.Interface,
	gvr schema.GroupVersionResource,
	name string,
	logger *slog.Logger,
	convert func(unstructured.Unstructured) (T, error),
	opts *GetOptions,
) (T, error) {
	var zero T
	if convert == nil {
		return zero, fmt.Errorf("GetResource: convert function must be provided")
	}
	if opts == nil {
		opts = NewGetOptions()
	}
	var ri dynamic.ResourceInterface
	ri = client.Resource(gvr)
	if opts.namespace != nil {
		ri = client.Resource(gvr).Namespace(*opts.namespace)
	}
	uobj, err := ri.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if logger != nil {
			logger.ErrorContext(ctx, "failed to get resource", slog.String("name", name), slog.String("resource", gvr.Resource), slog.Any("error", err))
		}
		return zero, fmt.Errorf("failed to retrieve %s '%s': %w", gvr.Resource, name, err)
	}
	converted, err := convert(*uobj)
	if err != nil {
		if logger != nil {
			logger.ErrorContext(ctx, "failed to convert resource", slog.String("name", name), slog.String("resource", gvr.Resource), slog.Any("error", err))
		}
		return zero, fmt.Errorf("failed to convert %s '%s': %w", gvr.Resource, name, err)
	}
	return converted, nil
}
