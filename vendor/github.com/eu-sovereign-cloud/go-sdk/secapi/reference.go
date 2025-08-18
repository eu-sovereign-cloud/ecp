package secapi

type Reference interface {
	TenantReference | WorkspaceReference
}

type ReferencedResource[Ref Reference, Data any] struct {
	Ref  Ref
	Data Data
}

type TenantReference struct {
	Tenant TenantID
	Name   string
}

type WorkspaceReference struct {
	Tenant    TenantID
	Workspace WorkspaceID
	Name      string
}

func validateTenantReference(tref TenantReference) error {
	if tref.Tenant == "" {
		return ErrNoMetatadaTenant
	}

	if tref.Name == "" {
		return ErrNoMetatadaName
	}

	return nil
}

func validateWorkspaceReference(wref WorkspaceReference) error {
	if wref.Tenant == "" {
		return ErrNoMetatadaTenant
	}

	if wref.Workspace == "" {
		return ErrNoMetatadaWorkspace
	}

	if wref.Name == "" {
		return ErrNoMetatadaName
	}

	return nil
}
