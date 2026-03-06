package plugin

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/scope"
)

// noDelay is an injectable delay function that returns instantly — for tests.
func noDelay() int { return 0 }

func newTestBlockStorage() *BlockStorage {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return newBlockStorageWithDelay(logger, noDelay)
}

func newBlockStorageDomain(name string, sizeGB int) *regional.BlockStorageDomain {
	return &regional.BlockStorageDomain{
		Metadata: regional.Metadata{
			CommonMetadata: model.CommonMetadata{Name: name},
			Scope:          scope.Scope{Tenant: "test-tenant", Workspace: "test-workspace"},
		},
		Spec: regional.BlockStorageSpec{SizeGB: sizeGB},
	}
}

func TestBlockStorage_Create(t *testing.T) {
	bs := newTestBlockStorage()
	resource := newBlockStorageDomain("my-bs", 100)

	err := bs.Create(context.Background(), resource)
	require.NoError(t, err)
}

func TestBlockStorage_Delete(t *testing.T) {
	bs := newTestBlockStorage()
	resource := newBlockStorageDomain("my-bs", 100)

	err := bs.Delete(context.Background(), resource)
	require.NoError(t, err)
}

func TestBlockStorage_IncreaseSize(t *testing.T) {
	bs := newTestBlockStorage()
	resource := newBlockStorageDomain("my-bs", 200)

	err := bs.IncreaseSize(context.Background(), resource)
	require.NoError(t, err)
}

func TestBlockStorage_IncreaseSize_DelayIsDoubled(t *testing.T) {
	callCount := 0
	countingDelay := func() int {
		callCount++
		return 0
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bs := newBlockStorageWithDelay(logger, countingDelay)
	resource := newBlockStorageDomain("my-bs", 200)

	err := bs.IncreaseSize(context.Background(), resource)
	require.NoError(t, err)
	assert.Equal(t, 2, callCount, "IncreaseSize should call delayFunc exactly twice")
}

func TestNewBlockStorage_UsesDefaultDelay(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bs := NewBlockStorage(logger)
	assert.NotNil(t, bs)
	assert.NotNil(t, bs.delayFunc)
}
