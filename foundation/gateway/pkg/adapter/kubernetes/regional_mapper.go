package kubernetes

import (
	storageskuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/block-storage/skus/v1"
	netowrkskuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/network/skus/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
)

func FromCRToNetworkSKUDomain(cr netowrkskuv1.NetworkSKU) *regional.NetworkSKUDomain {
	return &regional.NetworkSKUDomain{
		Metadata: model.Metadata{Name: cr.Name, Namespace: cr.Namespace},
		Spec: regional.NetworkSKUSpec{
			Bandwidth: cr.Spec.Bandwidth,
			Packets:   cr.Spec.Packets,
		},
	}
}

func FromCRToStorageSKUDomain(cr storageskuv1.StorageSKU) *regional.StorageSKUDomain {
	return &regional.StorageSKUDomain{
		Metadata: model.Metadata{
			Name: cr.Name,
		},
		Spec: regional.StorageSKUSpec{
			Iops:          int64(cr.Spec.Iops),
			MinVolumeSize: int64(cr.Spec.MinVolumeSize),
			Type:          string(cr.Spec.Type),
		},
	}
}
