package regional

// NOTE: Should base URLs and Provider names be passed at API deployment time?
// Base URLs definitely should, provider names could be retrieved from cluster (not sure if it's worth the effort).
const (
	WorkspaceBaseURL      = "/providers/seca.workspace"
	ProviderWorkspaceName = "seca.workspace/v1"
)

type WorkspaceDomain struct {
	Metadata

	Spec   WorkspaceSpec
	Status *WorkspaceStatusDomain
}

type WorkspaceSpec = map[string]interface{}

type WorkspaceStatusDomain struct {
	StatusDomain
	ResourceCount *int
}
