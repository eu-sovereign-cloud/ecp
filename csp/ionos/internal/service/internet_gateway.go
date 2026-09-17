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

// Update returns nil for the supported EgressOnly: false case: the gateway is a declaration
// with no backing IONOS object, and HandleUpdate hands the plugin the desired state on every
// reconcile of an active resource, so re-running Create here would announce the gateway ready
// again on each pass.
//
// Only when EgressOnly is true does Update delegate to Create, so that a spec change the store
// cannot honour is refused rather than silently kept Active. See InternetGatewayStore.Create.
func (i *InternetGateway) Update(ctx context.Context, resource *internetgatewaydom.InternetGateway) error {
	if !resource.Spec.EgressOnly {
		return nil
	}

	return i.Creator.Do(ctx, resource)
}

func (i *InternetGateway) Create(ctx context.Context, resource *internetgatewaydom.InternetGateway) error {
	return i.Creator.Do(ctx, resource)
}

func (i *InternetGateway) Delete(ctx context.Context, resource *internetgatewaydom.InternetGateway) error {
	return i.Deleter.Do(ctx, resource)
}
