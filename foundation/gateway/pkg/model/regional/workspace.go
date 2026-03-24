package regional

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
