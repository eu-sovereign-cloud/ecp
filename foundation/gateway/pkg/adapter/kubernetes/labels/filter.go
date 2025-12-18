package labels

import (
	"maps"
	"strings"
)

// FilterInternalLabels removes internal labels from the provided map.
func FilterInternalLabels(labels map[string]string) map[string]string {
	filteredLabels := maps.Clone(labels)
	maps.DeleteFunc(filteredLabels, func(k string, v string) bool {
		return strings.HasPrefix(k, InternalLabelPrefix)
	})
	return filteredLabels
}

// FilterKeyedLabels removes keyed labels from the provided map.
func FilterKeyedLabels(labels map[string]string) map[string]string {
	filteredLabels := maps.Clone(labels)
	maps.DeleteFunc(filteredLabels, func(k string, v string) bool {
		return strings.HasPrefix(k, KeyedLabelsPrefix)
	})
	return filteredLabels
}

// GetInternalLabels extracts internal labels from the provided map.
func GetInternalLabels(labels map[string]string) map[string]string {
	internalLabels := maps.Clone(labels)
	maps.DeleteFunc(internalLabels, func(k string, v string) bool {
		return !strings.HasPrefix(k, InternalLabelPrefix)
	})
	return internalLabels
}

// GetKeyedLabels extracts keyed labels from the provided map. Also trims the prefix.
func GetKeyedLabels(labels map[string]string) map[string]string {
	keyedLabels := maps.Clone(labels)
	maps.DeleteFunc(keyedLabels, func(k string, v string) bool {
		return !strings.HasPrefix(k, KeyedLabelsPrefix)
	})
	return keyedLabels
}

// GetCSPLabels returns labels set by the CSP by filtering out internal and keyed labels.
func GetCSPLabels(labels map[string]string) map[string]string {
	return FilterKeyedLabels(FilterInternalLabels(labels))
}
