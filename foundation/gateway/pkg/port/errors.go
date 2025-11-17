package port

import "errors"

var (
	Err                 = errors.New("port error")
	GettingResourceErr  = errors.Join(Err, errors.New("error getting resource"))
	ResourceNotFoundErr = errors.Join(Err, errors.New("resource not found"))
	UpdatingResourceErr = errors.Join(Err, errors.New("error updating resource"))
	CreatingResourceErr = errors.Join(Err, errors.New("error creating resource"))
	DeletingResourceErr = errors.Join(Err, errors.New("error deleting resource"))
)
