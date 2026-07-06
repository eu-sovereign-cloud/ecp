package rest

import (
	"testing"

	"github.com/stretchr/testify/require"

	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

func TestWorkspaceIteratorToAPI_ResponseMetadata(t *testing.T) {
	iter := workspaceIteratorToAPI(nil, nil)
	require.Equal(t, "workspaces", iter.Metadata.Resource)
	require.Equal(t, "seca.workspace/v1", iter.Metadata.Provider)
}

func TestWorkspaceToAPI_ResourceAndRef(t *testing.T) {
	ws := wsdom.Workspace{}
	ws.Name = "ws1"
	ws.Tenant = "t1"
	ws.Provider = wsdom.ProviderID

	out := workspaceToAPI(ws, "get")

	require.Equal(t, "workspace/ws1", out.Metadata.Resource)
	require.Equal(t, "seca.workspace/v1/tenants/t1/providers/workspace/ws1", out.Metadata.Ref)
}
