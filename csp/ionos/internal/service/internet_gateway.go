package service

import (
	"context"

	internetgatewayctrl "github.com/eu-sovereign-cloud/ecp/csp/ionos/internal/controller/internet_gateway"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
	internetgatewayk8s "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
)

var _ internetgatewayk8s.InternetGatewayPlugin = (*InternetGateway)(nil)

type InternetGateway struct {
	Creator *internetgatewayctrl.CreateInternetGateway
	Deleter *internetgatewayctrl.DeleteInternetGateway
}

// Update re-applies the desired state through the same path as Create, so a spec change the
// store cannot honour (Spec.EgressOnly flipped to true) is refused rather than silently kept
// Active. See InternetGatewayStore.Create.
func (i *InternetGateway) Update(ctx context.Context, resource *internetgatewaydom.InternetGateway) error {
	return i.Creator.Do(ctx, resource)
}

func (i *InternetGateway) Create(ctx context.Context, resource *internetgatewaydom.InternetGateway) error {
	return i.Creator.Do(ctx, resource)
}

func (i *InternetGateway) Delete(ctx context.Context, resource *internetgatewaydom.InternetGateway) error {
	return i.Deleter.Do(ctx, resource)
}
