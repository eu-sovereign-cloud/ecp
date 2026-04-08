package regional

type WorkspaceDomain struct {
	Metadata

	Spec   WorkspaceSpecDomain
	Status *WorkspaceStatusDomain
}

type WorkspaceSpecDomain = map[string]interface{}

type WorkspaceStatusDomain struct {
	StatusDomain
	ResourceCount *int
}
