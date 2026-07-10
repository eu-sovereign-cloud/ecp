package resource

// ListFilter is satisfied by any list-parameters type carrying pagination, filtering, and
// scope. ReaderRepo.List depends only on this interface (not the concrete ListParams struct),
// so a resource with an extra scoping dimension (e.g. Network) can carry it on its own local
// params type — implementing ListFilter via an embedded ListParams plus a GetXxx() method of
// its own — without adding a field to the struct every other resource shares.
type ListFilter interface {
	GetTenant() string
	GetWorkspace() string
	GetLimit() int
	GetSkipToken() string
	GetSelector() string
}

// ListParams carries pagination and filtering parameters for listing resources.
type ListParams struct {
	Scope

	Limit     int
	SkipToken string
	Selector  string
}

func (p ListParams) GetLimit() int        { return p.Limit }
func (p ListParams) GetSkipToken() string { return p.SkipToken }
func (p ListParams) GetSelector() string  { return p.Selector }

var _ ListFilter = ListParams{}
