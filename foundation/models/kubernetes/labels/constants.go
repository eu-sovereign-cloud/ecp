package labels

const (
	InternalLabelPrefix = "secapi.cloud/"
	KeyedLabelsPrefix   = "kl/"

	InternalProviderLabel  = InternalLabelPrefix + "provider"
	InternalRegionLabel    = InternalLabelPrefix + "region"
	InternalTenantLabel    = InternalLabelPrefix + "tenant"
	InternalWorkspaceLabel = InternalLabelPrefix + "workspace"
)
