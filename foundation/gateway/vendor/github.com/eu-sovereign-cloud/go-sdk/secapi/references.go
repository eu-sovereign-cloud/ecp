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

type NetworkReference struct {
	Tenant    TenantID
	Workspace WorkspaceID
	Network   NetworkID
	Name      string
}

// Validators

func (tref *TenantReference) validate() error {
	if tref.Tenant == "" {
		return ErrNoMetatadaTenant
	}

	if tref.Name == "" {
		return ErrNoMetatadaName
	}

	return nil
}

func (wref *WorkspaceReference) validate() error {
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

func (nref *NetworkReference) validate() error {
	if nref.Tenant == "" {
		return ErrNoMetatadaTenant
	}

	if nref.Workspace == "" {
		return ErrNoMetatadaWorkspace
	}

	if nref.Network == "" {
		return ErrNoPathMetadata
	}

	if nref.Name == "" {
		return ErrNoMetatadaName
	}

	return nil
}
