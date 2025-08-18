package secapi

import "errors"

var (
	ErrNoMetatada          = errors.New("metadata is empty")
	ErrNoMetatadaTenant    = errors.New("metadata tenant is empty")
	ErrNoMetatadaWorkspace = errors.New("metadata workspace is empty")
	ErrNoMetatadaName      = errors.New("metadata name is empty")
)
