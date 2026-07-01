// Package rest provides REST↔domain conversion and HTTP handlers for the workspace resource.
package rest

import (
	"fmt"
	"net/http"
	"strconv"

	sdkworkspace "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

	"github.com/eu-sovereign-cloud/ecp/framework/kernel/port/persistence"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	"github.com/eu-sovereign-cloud/ecp/framework/kernel/validation"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	commonfrontend "github.com/eu-sovereign-cloud/ecp/resource/common/frontend"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

const (
	// WorkspaceAPIVersion is the API version string used in response metadata.
	WorkspaceAPIVersion = wsdom.Version
	// WorkspaceResource is the resource name.
	WorkspaceResource = wsdom.Resource
)

// WorkspaceIdentity carries identity for a single workspace resource.
type WorkspaceIdentity struct {
	name            string
	tenant          string
	resourceVersion string
}

func (w *WorkspaceIdentity) GetName() string      { return w.name }
func (w *WorkspaceIdentity) GetVersion() string   { return w.resourceVersion }
func (w *WorkspaceIdentity) GetTenant() string    { return w.tenant }
func (w *WorkspaceIdentity) GetWorkspace() string { return "" }

var _ persistence.IdentifiableResource = (*WorkspaceIdentity)(nil)

// listParamsFromAPI converts SDK ListWorkspacesParams to resource.ListParams.
func listParamsFromAPI(params sdkworkspace.ListWorkspacesParams, tenant string) resource.ListParams {
	limit := validation.GetLimit(params.Limit)

	var skipToken string
	if params.SkipToken != nil {
		skipToken = *params.SkipToken
	}

	var selector string
	if params.Labels != nil {
		selector = *params.Labels
	}

	return resource.ListParams{
		Scope: resource.Scope{
			Tenant: tenant,
		},
		Limit:     limit,
		SkipToken: skipToken,
		Selector:  selector,
	}
}

// workspaceToAPIWithVerb returns a func that converts a Workspace to its SDK representation with the given verb.
func workspaceToAPIWithVerb(verb string) func(ws *wsdom.Workspace) *sdkschema.Workspace {
	return func(ws *wsdom.Workspace) *sdkschema.Workspace {
		return workspaceToAPI(*ws, verb)
	}
}

// workspaceIteratorToAPI converts a list of Workspace to an SDK WorkspaceIterator.
func workspaceIteratorToAPI(wss []*wsdom.Workspace, nextSkipToken *string) *sdkworkspace.WorkspaceIterator {
	items := make([]sdkschema.Workspace, len(wss))
	for i, ws := range wss {
		items[i] = *(workspaceToAPI(*ws, http.MethodGet))
	}

	iterator := &sdkworkspace.WorkspaceIterator{
		Items: items,
		Metadata: sdkschema.ResponseMetadata{
			Provider: wsdom.ProviderID,
			Resource: WorkspaceResource,
			Verb:     http.MethodGet,
		},
	}

	if nextSkipToken != nil {
		iterator.Metadata.SkipToken = nextSkipToken
	}

	return iterator
}

// workspaceToAPI converts a Workspace to a schema.Workspace with the given verb.
func workspaceToAPI(ws wsdom.Workspace, verb string) *sdkschema.Workspace {
	resourceVersion := int64(0)
	if parsed, err := strconv.ParseInt(ws.ResourceVersion, 10, 64); err == nil {
		resourceVersion = parsed
	}

	sdk := &sdkschema.Workspace{
		Metadata: &sdkschema.RegionalResourceMetadata{
			ApiVersion:      WorkspaceAPIVersion,
			CreatedAt:       ws.CreatedAt,
			LastModifiedAt:  ws.UpdatedAt,
			Kind:            sdkschema.RegionalResourceMetadataKindResourceKindWorkspace,
			Name:            ws.Name,
			Tenant:          ws.Tenant,
			Provider:        ws.Provider,
			Region:          ws.Region,
			Resource:        fmt.Sprintf(commondomain.RegionalResourceFormat, sdkschema.RegionalResourceMetadataKindResourceKindWorkspace, ws.Name),
			Ref:             fmt.Sprintf(ws.Provider+"/"+commondomain.RegionalTenantScopedResourceFormat, ws.Tenant, sdkschema.RegionalResourceMetadataKindResourceKindWorkspace, ws.Name),
			ResourceVersion: resourceVersion,
			Verb:            verb,
		},
		Labels:      ws.Labels,
		Annotations: ws.Annotations,
		Extensions:  ws.Extensions,
		Spec:        ws.Spec,
	}
	if sdk.Labels == nil {
		sdk.Labels = make(sdkschema.Labels)
	}
	if ws.Status != nil {
		sdk.Status = &sdkschema.WorkspaceStatus{
			ResourceCount: ws.Status.ResourceCount,
			State:         commonfrontend.ResourceStateToAPI(ws.Status.State),
			Conditions:    commonfrontend.ConditionsToAPI(ws.Status.Conditions),
		}
	}
	if ws.DeletedAt != nil {
		sdk.Metadata.DeletedAt = ws.DeletedAt
	}
	return sdk
}

// workspaceFromAPI converts an SDK Workspace to a Workspace.
func workspaceFromAPI(api sdkschema.Workspace, id *WorkspaceIdentity, region string) *wsdom.Workspace {
	ws := &wsdom.Workspace{
		Spec: api.Spec,
	}
	ws.Name = id.GetName()
	ws.ResourceVersion = id.GetVersion()
	ws.Provider = wsdom.ProviderID
	ws.Tenant = id.GetTenant()
	ws.Region = region
	ws.Labels = api.Labels
	ws.Annotations = api.Annotations
	ws.Extensions = api.Extensions

	return ws
}
