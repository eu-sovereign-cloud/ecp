package service

import (
	"context"

	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// The Ionos plugin does not implement updates yet, and reports that by doing nothing.
//
// ErrNotSupported would be wrong here even though nothing is applied. Update is level-triggered -
// it runs on every reconcile of an active resource, not only after an edit - and the plugin has no
// observed state to diff against, so it cannot tell a resource nobody touched from one carrying a
// change it must refuse. Returning ErrNotSupported unconditionally therefore stamps an UpdateFailed
// condition on every healthy Ionos-backed resource for an update that was never requested, which
// leaves the condition meaning nothing at all: a reader can no longer tell "nothing to do" from
// "refused". ErrNotSupported belongs on a diff the plugin has actually detected and will not apply.
//
// The cost is that a real edit is dropped silently. That is the weaker failure of the two, and it
// is documented per-provider in doc/PLUGINS.md rather than reported per-resource.

func (s *Network) Update(_ context.Context, _ *netdom.Network) error { return nil }

func (s *Workspace) Update(_ context.Context, _ *wsdom.Workspace) error { return nil }

func (s *BlockStorage) Update(_ context.Context, _ *bsdom.BlockStorage) error { return nil }
