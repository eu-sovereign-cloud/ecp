package regional

import (
	skuv1 "github.com/eu-sovereign-cloud/ecp/foundation/api/network/skus/v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
)

func FromUnstructuredToNetworkSKUDomain(u unstructured.Unstructured) (*NetworkSKUDomain, error) {
	var cr skuv1.NetworkSKU
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &cr); err != nil {
		return nil, err
	}
	return FromCRToNetworkSKUDomain(cr), nil
}

func FromCRToNetworkSKUDomain(cr skuv1.NetworkSKU) *NetworkSKUDomain {
	return &NetworkSKUDomain{
		Metadata: model.Metadata{Name: cr.Name, Namespace: cr.Namespace},
		Spec: NetworkSKUSpec{
			Bandwidth: cr.Spec.Bandwidth,
			Packets:   cr.Spec.Packets,
		},
	}
}

func ToSDKNetworkSKU(domain *NetworkSKUDomain) *sdkschema.NetworkSku {
	return &sdkschema.NetworkSku{
		Metadata: &sdkschema.SkuResourceMetadata{Name: domain.Name},
		Spec: &sdkschema.NetworkSkuSpec{
			Bandwidth: domain.Spec.Bandwidth,
			Packets:   domain.Spec.Packets,
		},
	}
}
