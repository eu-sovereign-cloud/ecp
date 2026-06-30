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
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
)

// NicIdentity carries identity for a single NIC resource.
type NicIdentity struct {
	name            string
	tenant          string
	workspace       string
	resourceVersion string
}

func (n *NicIdentity) GetName() string      { return n.name }
func (n *NicIdentity) GetVersion() string   { return n.resourceVersion }
func (n *NicIdentity) GetTenant() string    { return n.tenant }
func (n *NicIdentity) GetWorkspace() string { return n.workspace }

var _ persistence.IdentifiableResource = (*NicIdentity)(nil)

// nicListParamsFromAPI converts SDK ListNicsParams to resource.ListParams.
func nicListParamsFromAPI(params sdknetwork.ListNicsParams, tenant, workspace string) resource.ListParams {
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

// nicToAPIWithVerb returns a func that converts a Nic to its SDK representation with the given verb.
func nicToAPIWithVerb(verb string) func(n *nicdom.Nic) *sdkschema.Nic {
	return func(n *nicdom.Nic) *sdkschema.Nic {
		sdk := nicToAPI(n)
		sdk.Metadata.Verb = verb
		return sdk
	}
}

func nicToAPI(n *nicdom.Nic) *sdkschema.Nic {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(n.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	const kind = sdkschema.RegionalWorkspaceResourceMetadataKindResourceKindNic

	out := &sdkschema.Nic{
		Metadata: &sdkschema.RegionalWorkspaceResourceMetadata{
			ApiVersion:     nicdom.Version,
			CreatedAt:      n.CreatedAt,
			LastModifiedAt: n.UpdatedAt,
			Kind:           kind,
			Name:           n.Name,
			Tenant:         n.Tenant,
			Workspace:      n.Workspace,
			Provider:       n.Provider,
			Region:         n.Region,
			Resource:       fmt.Sprintf(commondomain.RegionalResourceFormat, kind, n.Name),
			Ref: fmt.Sprintf(
				n.Provider+"/"+commondomain.RegionalWorkspaceScopedResourceFormat,
				n.Tenant, n.Workspace, kind, n.Name,
			),
			ResourceVersion: resourceVersion,
		},
		Labels:      n.Labels,
		Annotations: n.Annotations,
		Extensions:  n.Extensions,
		Spec: sdkschema.NicSpec{
			Addresses: n.Spec.Addresses,
			SubnetRef: commonfrontend.ReferenceToAPI(n.Spec.SubnetRef),
		},
	}

	if out.Labels == nil {
		out.Labels = make(sdkschema.Labels)
	}
	if n.Spec.SkuRef != (commondomain.Reference{}) {
		ref := commonfrontend.ReferenceToAPI(n.Spec.SkuRef)
		out.Spec.SkuRef = &ref
	}
	for _, r := range n.Spec.PublicIpRefs {
		out.Spec.PublicIpRefs = append(out.Spec.PublicIpRefs, commonfrontend.ReferenceToAPI(r))
	}
	for _, r := range n.Spec.SecurityGroupRefs {
		out.Spec.SecurityGroupRefs = append(out.Spec.SecurityGroupRefs, commonfrontend.ReferenceToAPI(r))
	}

	if n.Status != nil {
		out.Status = &sdkschema.NicStatus{
			Conditions: commonfrontend.ConditionsToAPI(n.Status.Conditions),
			State:      commonfrontend.ResourceStateToAPI(n.Status.State),
			MacAddress: n.Status.MacAddress,
			Addresses:  n.Status.Addresses,
		}
		for _, r := range n.Status.PublicIpRefs {
			out.Status.PublicIpRefs = append(out.Status.PublicIpRefs, commonfrontend.ReferenceToAPI(r))
		}
	}
	if n.DeletedAt != nil {
		out.Metadata.DeletedAt = n.DeletedAt
	}
	return out
}

// nicIteratorToAPI converts a list of Nic to an SDK NicIterator.
func nicIteratorToAPI(ns []*nicdom.Nic, nextSkipToken *string) *sdknetwork.NicIterator {
	items := make([]sdkschema.Nic, len(ns))
	for i := range ns {
		items[i] = *nicToAPI(ns[i])
	}
	iterator := &sdknetwork.NicIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: nicdom.ProviderID,
			Resource: nicdom.Resource,
			Verb:     "list",
		},
	}
	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}
	return iterator
}

// nicFromAPI converts an SDK Nic to a Nic.
func nicFromAPI(sdk sdkschema.Nic, id *NicIdentity, region string) *nicdom.Nic {
	n := &nicdom.Nic{
		Spec: nicdom.NicSpec{
			Addresses: sdk.Spec.Addresses,
			SubnetRef: commonfrontend.ReferenceFromAPI(sdk.Spec.SubnetRef),
		},
	}
	if sdk.Spec.SkuRef != nil {
		n.Spec.SkuRef = commonfrontend.ReferenceFromAPI(*sdk.Spec.SkuRef)
	}
	for _, r := range sdk.Spec.PublicIpRefs {
		n.Spec.PublicIpRefs = append(n.Spec.PublicIpRefs, commonfrontend.ReferenceFromAPI(r))
	}
	for _, r := range sdk.Spec.SecurityGroupRefs {
		n.Spec.SecurityGroupRefs = append(n.Spec.SecurityGroupRefs, commonfrontend.ReferenceFromAPI(r))
	}

	n.Name = id.GetName()
	n.ResourceVersion = id.GetVersion()
	n.Provider = nicdom.ProviderID
	n.Tenant = id.GetTenant()
	n.Workspace = id.GetWorkspace()
	n.Region = region
	n.Labels = sdk.Labels
	n.Annotations = sdk.Annotations
	n.Extensions = sdk.Extensions

	return n
}

// newNicWithIdentity returns a *nicdom.Nic populated with identity fields from ir.
func newNicWithIdentity(ir persistence.IdentifiableResource) *nicdom.Nic {
	d := &nicdom.Nic{}
	d.Name = ir.GetName()
	d.Tenant = ir.GetTenant()
	d.Workspace = ir.GetWorkspace()
	d.ResourceVersion = ir.GetVersion()
	return d
}
