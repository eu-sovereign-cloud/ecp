package labels

import "strings"

const (
	InternalLabelPrefix = "internal/"
	KeyedLabelsPrefix   = "kl/"

	InternalProviderLabel  = InternalLabelPrefix + "provider"
	InternalRegionLabel    = InternalLabelPrefix + "region"
	InternalTenantLabel    = InternalLabelPrefix + "tenant"
	InternalWorkspaceLabel = InternalLabelPrefix + "workspace"
)

// FilterInternalLabels removes internal labels from the provided map.
func FilterInternalLabels(labels map[string]string) map[string]string {
	filteredLabels := make(map[string]string)
	for k, v := range labels {
		if strings.HasPrefix(k, InternalLabelPrefix) {
			continue
		}
		filteredLabels[k] = v
	}
	return filteredLabels
}

// FilterKeyedLabels removes keyed labels from the provided map.
func FilterKeyedLabels(labels map[string]string) map[string]string {
	filteredLabels := make(map[string]string)
	for k, v := range labels {
		if strings.HasPrefix(k, KeyedLabelsPrefix) {
			continue
		}
		filteredLabels[k] = v
	}
	return filteredLabels
}

// GetInternalLabels extracts internal labels from the provided map.
func GetInternalLabels(labels map[string]string) map[string]string {
	internalLabels := make(map[string]string)
	for k, v := range labels {
		if strings.HasPrefix(k, InternalLabelPrefix) {
			internalLabels[k] = v
		}
	}
	return internalLabels
}

// GetKeyedLabels extracts keyed labels from the provided map. Also trims the prefix.
func GetKeyedLabels(labels map[string]string) map[string]string {
	keyedLabels := make(map[string]string)
	for k, v := range labels {
		if strings.HasPrefix(k, KeyedLabelsPrefix) {
			keyedLabels[strings.TrimPrefix(k, KeyedLabelsPrefix)] = v
		}
	}
	return keyedLabels
}

// GetCSPLabels returns labels set by the CSP by filtering out internal and keyed labels.
func GetCSPLabels(labels map[string]string) map[string]string {
	return FilterKeyedLabels(FilterInternalLabels(labels))
}
