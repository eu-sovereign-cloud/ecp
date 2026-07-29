package rest

import (
	"fmt"
	"strconv"

	sdknetwork "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.network.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resource/common/frontend"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
)

// publicIpListParamsFromAPI converts SDK ListPublicIpsParams to resource.ListParams.
func publicIpListParamsFromAPI(params sdknetwork.ListPublicIpsParams, tenant, workspace string) resource.ListParams {
	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}
	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}
	return resource.ListParams{
		Scope:     resource.Scope{Tenant: tenant, Workspace: workspace},
		Limit:     validation.GetLimit(params.Limit),
		SkipToken: skipToken,
		Selector:  selector,
	}
}

// publicIpToAPIWithVerb returns a func that converts a PublicIp to its SDK representation with the given verb.
func publicIpToAPIWithVerb(verb string) func(p *publicipdom.PublicIp) *sdkschema.PublicIp {
	return func(p *publicipdom.PublicIp) *sdkschema.PublicIp {
		sdk := publicIpToAPI(p)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

func publicIpToAPI(p *publicipdom.PublicIp) *sdkschema.PublicIp {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(p.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	const kind = sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindPublicIP

	out := &sdkschema.PublicIp{
		Metadata: &sdkschema.RegionalWorkspaceResourceMetadata{
			ApiVersion:     publicipdom.Version,
			CreatedAt:      p.CreatedAt,
			LastModifiedAt: p.UpdatedAt,
			Kind:           kind,
			Name:           p.Name,
			Tenant:         p.Tenant,
			Workspace:      p.Workspace,
			Provider:       p.Provider,
			Region:         p.Region,
			Resource:       fmt.Sprintf(commondomain.RegionalResourceFormat, kind, p.Name),
			Ref: fmt.Sprintf(
				p.Provider+"/"+commondomain.RegionalWorkspaceScopedResourceFormat,
				p.Tenant, p.Workspace, kind, p.Name,
			),
			ResourceVersion: resourceVersion,
		},
		Labels:      p.Labels,
		Annotations: p.Annotations,
		Extensions:  p.Extensions,
		Spec: sdkschema.PublicIpSpec{
			Address: p.Spec.Address,
			Version: commonfrontend.IPVersionToAPI(p.Spec.Version),
		},
	}

	if out.Labels == nil {
		out.Labels = make(sdkschema.Labels)
	}

	if p.Status != nil {
		out.Status = &sdkschema.PublicIpStatus{
			Conditions: commonfrontend.ConditionsToAPI(p.Status.Conditions),
			State:      commonfrontend.ResourceStateToAPI(p.Status.State),
			IpAddress:  p.Status.IpAddress,
			AttachedTo: commonfrontend.ReferencePtrToAPI(p.Status.AttachedTo),
		}
	}
	if p.DeletedAt != nil {
		out.Metadata.DeletedAt = p.DeletedAt
	}
	return out
}

// publicIpIteratorToAPI converts a list of PublicIp to an SDK PublicIpIterator.
func publicIpIteratorToAPI(ps []*publicipdom.PublicIp, nextSkipToken *string) *sdknetwork.PublicIpIterator {
	items := make([]sdkschema.PublicIp, len(ps))
	for i := range ps {
		items[i] = *publicIpToAPI(ps[i])
	}
	iterator := &sdknetwork.PublicIpIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: publicipdom.ProviderID,
			Resource: publicipdom.Resource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return iterator
}

// publicIpFromAPI converts an SDK PublicIp to a PublicIp.
func publicIpFromAPI(sdk sdkschema.PublicIp, id *resource.Identity, region string) *publicipdom.PublicIp {
	p := &publicipdom.PublicIp{
		Spec: publicipdom.PublicIpSpec{
			Address: sdk.Spec.Address,
			Version: commonfrontend.IPVersionFromAPI(sdk.Spec.Version),
		},
	}

	p.Name = id.GetName()
	p.ResourceVersion = id.GetVersion()
	p.Provider = publicipdom.ProviderID
	p.Tenant = id.GetTenant()
	p.Workspace = id.GetWorkspace()
	p.Region = region
	p.Labels = sdk.Labels
	p.Annotations = sdk.Annotations
	p.Extensions = sdk.Extensions

	return p
}

// newPublicIpWithIdentity returns a *publicipdom.PublicIp populated with identity fields from ir.
func newPublicIpWithIdentity(ir persistence.IdentifiableResource) *publicipdom.PublicIp {
	d := &publicipdom.PublicIp{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	d.Workspace = ir.GetWorkspace()
	d.ResourceVersion = ir.GetVersion()
	return d
}
