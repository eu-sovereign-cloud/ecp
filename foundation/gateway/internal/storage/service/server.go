package service

import storagev1 "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.storage.v1"

var _ storagev1.ServerInterface = (*SKUService)(nil)

type Service struct {
	GetSku
	// ...
}
