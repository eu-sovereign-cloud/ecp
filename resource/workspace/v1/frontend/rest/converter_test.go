package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

func TestWorkspaceIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := workspaceIteratorToAPI(nil, nil)
	// ResponseMetadata.resource: {collection}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "workspaces", iter.Metadata.Resource)
	require.Equal(t, "seca.workspace/v1", iter.Metadata.Provider)
}

func TestWorkspaceToAPI_ResourceAndRef(t *testing.T) {
	ws := wsdom.Workspace{}
	ws.Name = "ws1"
	ws.Tenant = "t1"
	ws.Provider = wsdom.ProviderID

	out := workspaceToAPI(ws, "get")

	// metadata.resource: {collection}/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "workspaces/ws1", out.Metadata.Resource)
	// metadata.ref: {provider}/tenants/{tenant}/workspaces/{name}
	// Spec: https://spec.secapi.cloud/docs/content/Architecture/resource-model#metadata
	require.Equal(t, "seca.workspace/v1/tenants/t1/workspaces/ws1", out.Metadata.Ref)
}
