//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/wait"

	kernel "github.com/eu-sovereign-cloud/ecp/framework/kernel"
	kernelresource "github.com/eu-sovereign-cloud/ecp/framework/kernel/resource"
	commondomain "github.com/eu-sovereign-cloud/ecp/resource/common/domain"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
)

// newBlockStorage builds a block storage in the test workspace, optionally created from a source image.
func newBlockStorage(name string, sourceImageRef *commondomain.Reference) *bsdom.BlockStorage {
	return &bsdom.BlockStorage{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: name},
			Scope:          kernelresource.Scope{Tenant: testTenant, Workspace: testWorkspace},
		},
		Spec: bsdom.BlockStorageSpec{
			SizeGB:         1,
			SkuRef:         commondomain.Reference{Resource: "sku-1"},
			SourceImageRef: sourceImageRef,
		},
	}
}

// newImage builds a tenant-scoped image stored on the block storage identified by blockStorageRef.
func newImage(name string, blockStorageRef commondomain.Reference) *imgdom.Image {
	return &imgdom.Image{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: name},
			Scope:          kernelresource.Scope{Tenant: testTenant},
		},
		Spec: imgdom.ImageSpec{
			BlockStorageRef: blockStorageRef,
			CpuArchitecture: "amd64",
			Boot:            "UEFI",
			Initializer:     "none",
		},
	}
}

// blockStorageRefFor returns an image's reference to a workspace-scoped block storage by name.
func blockStorageRefFor(name string) commondomain.Reference {
	return commondomain.Reference{Workspace: testWorkspace, Resource: "block-storages/" + name}
}

// imageRefFor returns a block storage's source reference to a tenant-scoped image by name.
func imageRefFor(name string) commondomain.Reference {
	return commondomain.Reference{Resource: "images/" + name}
}

func loadBlockStorage(ctx context.Context, name string) (*bsdom.BlockStorage, error) {
	bs := &bsdom.BlockStorage{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: name},
			Scope:          kernelresource.Scope{Tenant: testTenant, Workspace: testWorkspace},
		},
	}
	err := blockStorageRepo.Load(ctx, &bs)
	return bs, err
}

func loadImage(ctx context.Context, name string) (*imgdom.Image, error) {
	img := &imgdom.Image{
		RegionalMetadata: commondomain.RegionalMetadata{
			CommonMetadata: commondomain.CommonMetadata{Name: name},
			Scope:          kernelresource.Scope{Tenant: testTenant},
		},
	}
	err := imageRepo.Load(ctx, &img)
	return img, err
}

// createActiveBlockStorage creates a block storage and waits until it becomes active.
func createActiveBlockStorage(t *testing.T, name string) *bsdom.BlockStorage {
	t.Helper()

	bs := newBlockStorage(name, nil)
	_, err := blockStorageRepo.Create(t.Context(), bs)
	require.NoError(t, err)

	err = wait.PollUntilContextTimeout(t.Context(), pollInterval, dependencyTimeout, true, func(ctx context.Context) (bool, error) {
		loaded, err := loadBlockStorage(ctx, name)
		if err != nil {
			return false, err
		}
		return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
	})
	require.NoError(t, err, "block storage %s should become active", name)

	return bs
}

// createActiveImage creates an image stored on the named block storage and waits until it becomes active.
func createActiveImage(t *testing.T, name, blockStorageName string) *imgdom.Image {
	t.Helper()

	img := newImage(name, blockStorageRefFor(blockStorageName))
	_, err := imageRepo.Create(t.Context(), img)
	require.NoError(t, err)

	err = wait.PollUntilContextTimeout(t.Context(), pollInterval, dependencyTimeout, true, func(ctx context.Context) (bool, error) {
		loaded, err := loadImage(ctx, name)
		if err != nil {
			return false, err
		}
		return loaded.Status != nil && loaded.Status.State == commondomain.ResourceStateActive, nil
	})
	require.NoError(t, err, "image %s should become active", name)

	return img
}

func requireBlockStorageDeleted(t *testing.T, name string) {
	t.Helper()

	err := wait.PollUntilContextTimeout(t.Context(), pollInterval, dependencyTimeout, true, func(ctx context.Context) (bool, error) {
		if _, err := loadBlockStorage(ctx, name); err != nil {
			if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	require.NoError(t, err, "block storage %s should be deleted", name)
}

func requireImageDeleted(t *testing.T, name string) {
	t.Helper()

	err := wait.PollUntilContextTimeout(t.Context(), pollInterval, dependencyTimeout, true, func(ctx context.Context) (bool, error) {
		if _, err := loadImage(ctx, name); err != nil {
			if domainErr := kernel.AsError(err); domainErr != nil && domainErr.Kind == kernel.KindNotFound {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	require.NoError(t, err, "image %s should be deleted", name)
}

// hasConditionType reports whether conds contains a condition with the given type.
func hasConditionType(conds []commondomain.StatusCondition, conditionType string) bool {
	for _, c := range conds {
		if c.Type == conditionType {
			return true
		}
	}
	return false
}
