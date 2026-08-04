package service

import (
	"context"
	"fmt"

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// The Ionos plugin does not implement updates yet. Reporting that through ErrNotSupported is the
// honest answer: the reason lands on the resource's status where a user can see it, and the
// reconciler stops rather than retrying an operation that will not start working on its own.
// Returning nil instead would claim the change had been applied when nothing happened.
var errUpdateUnimplemented = fmt.Errorf("%w: the Ionos plugin does not implement updates", backendport.ErrNotSupported)

func (s *Network) Update(_ context.Context, _ *netdom.Network) error { return errUpdateUnimplemented }

func (s *Workspace) Update(_ context.Context, _ *wsdom.Workspace) error {
	return errUpdateUnimplemented
}

func (s *BlockStorage) Update(_ context.Context, _ *bsdom.BlockStorage) error {
	return errUpdateUnimplemented
}
