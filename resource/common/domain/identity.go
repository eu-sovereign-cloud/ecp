package domain

import "k8s.io/apimachinery/pkg/runtime/schema"

// GVR builds a Kubernetes GroupVersionResource from a slice's identity constants.
func GVR(group, version, resource string) schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
}

// GVK builds a Kubernetes GroupVersionKind from a slice's identity constants.
func GVK(group, version, kind string) schema.GroupVersionKind {
	return schema.GroupVersionKind{Group: group, Version: version, Kind: kind}
}
