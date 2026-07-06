// Package rest provides REST↔domain conversion and HTTP handlers for the workspace resource.
package rest

import (
	"fmt"
	"net/http"
	"strconv"

	sdkworkspace "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.workspace.v1"
	sdkschema "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/schema"

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
	// ResourceFormat formats a resource path string.
	ResourceFormat = "%s/%s"
	// TenantScopedResourceFormat formats a tenant-scoped resource ref.
	TenantScopedResourceFormat = "tenants/%s/providers/%s/%s"
)

// workspaceIdentity builds a RegionalMetadata carrying just enough identity to look up or delete a workspace resource.
func workspaceIdentity(name, tenant, resourceVersion string) *commondomain.RegionalMetadata {
	return &commondomain.RegionalMetadata{
		CommonMetadata: commondomain.CommonMetadata{Name: name, ResourceVersion: resourceVersion},
		Scope:          resource.Scope{Tenant: tenant},
	}
}

// ListParamsFromAPI converts SDK ListWorkspacesParams to resource.ListParams.
func ListParamsFromAPI(params sdkworkspace.ListWorkspacesParams, tenant string) resource.ListParams {
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

// WorkspaceToAPIWithVerb returns a func that converts a Workspace to its SDK representation with the given verb.
func WorkspaceToAPIWithVerb(verb string) func(ws *wsdom.Workspace) *sdkschema.Workspace {
	return func(ws *wsdom.Workspace) *sdkschema.Workspace {
		return workspaceToAPI(*ws, verb)
	}
}

// WorkspaceIteratorToAPI converts a list of Workspace to an SDK WorkspaceIterator.
func WorkspaceIteratorToAPI(wss []*wsdom.Workspace, nextSkipToken *string) *sdkworkspace.WorkspaceIterator {
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
			Resource:        fmt.Sprintf(ResourceFormat, sdkschema.RegionalResourceMetadataKindResourceKindWorkspace, ws.Name),
			Ref:             fmt.Sprintf(ws.Provider+"/"+TenantScopedResourceFormat, ws.Tenant, sdkschema.RegionalResourceMetadataKindResourceKindWorkspace, ws.Name),
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

// WorkspaceFromAPI converts an SDK Workspace to a Workspace.
func WorkspaceFromAPI(api sdkschema.Workspace, id *commondomain.RegionalMetadata, region string) *wsdom.Workspace {
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
