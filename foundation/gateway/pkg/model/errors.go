package model

import "errors"

// Sentinel errors for resource model operations
var (
	ErrLoad     = errors.New("load error")
	ErrWatch    = errors.New("watch error")
	ErrList     = errors.New("list error")
	ErrConvert  = errors.New("conversion error")
	ErrFilter   = errors.New("filtering error")
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)
