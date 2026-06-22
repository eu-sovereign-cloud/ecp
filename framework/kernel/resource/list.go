package resource

// ListParams carries pagination and filtering parameters for listing resources.
type ListParams struct {
	Scope

	Limit     int
	SkipToken string
	Selector  string
}
