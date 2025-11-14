package model

type ListParams struct {
	Namespace string
	Limit     int
	SkipToken string
	Selector  string
}
