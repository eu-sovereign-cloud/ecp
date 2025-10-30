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

// ListOptions provides a builder-style configuration for ListResources.
type ListOptions struct {
	namespace *string
	limit     int
	skipToken *string
	selector  *string
	logger    *slog.Logger
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
func (o *ListOptions) Logger(l *slog.Logger) *ListOptions { o.logger = l; return o }

// GetOptions provides a builder-style configuration for GetResource.
type GetOptions struct {
	namespace *string
	logger    *slog.Logger
}

func NewGetOptions() *GetOptions                        { return &GetOptions{} }
func (o *GetOptions) Namespace(ns string) *GetOptions   { o.namespace = &ns; return o }
func (o *GetOptions) Logger(l *slog.Logger) *GetOptions { o.logger = l; return o }

// ListResources lists Kubernetes custom resources applying builder options, label filtering, pagination and conversion.
func ListResources[T any](
	ctx context.Context,
	client dynamic.Interface,
	gvr schema.GroupVersionResource,
	convert func(unstructured.Unstructured) (T, error),
	opts *ListOptions,
) ([]T, *string, error) {
	if convert == nil {
		return nil, nil, fmt.Errorf("ListResources: convert function must be provided")
	}
	if opts == nil {
		opts = NewListOptions()
	}
	listOpts := metav1.ListOptions{}
	if opts.limit > 0 {
		listOpts.Limit = int64(opts.limit)
	}
	if opts.skipToken != nil {
		listOpts.Continue = *opts.skipToken
	}
	selectorStr := ""
	if opts.selector != nil {
		selectorStr = *opts.selector
		listOpts.LabelSelector = filter.K8sSelectorForAPI(selectorStr)
	}
	var ri dynamic.ResourceInterface
	if opts.namespace != nil {
		ri = client.Resource(gvr).Namespace(*opts.namespace)
	} else {
		ri = client.Resource(gvr)
	}
	ulist, err := ri.List(ctx, listOpts)
	if err != nil {
		if opts.logger != nil {
			opts.logger.ErrorContext(ctx, "failed to list resources", slog.String("resource", gvr.Resource), slog.Any("error", err))
		}
		return nil, nil, fmt.Errorf("failed to list resources for %s: %w", gvr.Resource, err)
	}
	res := make([]T, 0, len(ulist.Items))
	for _, item := range ulist.Items {
		if selectorStr != "" {
			match, k8sHandled, e := filter.MatchLabels(item.GetLabels(), selectorStr)
			if e != nil {
				if opts.logger != nil {
					opts.logger.WarnContext(ctx, "invalid selector, skipping", slog.String("resource", gvr.Resource), slog.String("selector", selectorStr), slog.Any("error", e))
				}
				continue
			}
			if !match && !k8sHandled {
				continue
			}
		}
		converted, e := convert(item)
		if e != nil {
			if opts.logger != nil {
				opts.logger.ErrorContext(ctx, "conversion failed", slog.String("resource", gvr.Resource), slog.Any("error", e))
			}
			return nil, nil, fmt.Errorf("failed to convert %s: %w", gvr.Resource, e)
		}
		res = append(res, converted)
	}
	next := ulist.GetContinue()
	if next == "" {
		return res, nil, nil
	}
	return res, &next, nil
}

// GetResource fetches and converts a single resource using builder options.
func GetResource[T any](
	ctx context.Context,
	client dynamic.Interface,
	gvr schema.GroupVersionResource,
	name string,
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
	if opts.namespace != nil {
		ri = client.Resource(gvr).Namespace(*opts.namespace)
	} else {
		ri = client.Resource(gvr)
	}
	uobj, err := ri.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if opts.logger != nil {
			opts.logger.ErrorContext(ctx, "failed to get resource", slog.String("name", name), slog.String("resource", gvr.Resource), slog.Any("error", err))
		}
		return zero, fmt.Errorf("failed to retrieve %s '%s': %w", gvr.Resource, name, err)
	}
	converted, err := convert(*uobj)
	if err != nil {
		if opts.logger != nil {
			opts.logger.ErrorContext(ctx, "failed to convert resource", slog.String("name", name), slog.String("resource", gvr.Resource), slog.Any("error", err))
		}
		return zero, fmt.Errorf("failed to convert %s '%s': %w", gvr.Resource, name, err)
	}
	return converted, nil
}
