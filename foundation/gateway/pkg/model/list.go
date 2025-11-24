package model

// ListParams - parameters for listing resources
type ListParams struct {
	Namespace string
	Limit     int
	SkipToken string
	Selector  string
}
