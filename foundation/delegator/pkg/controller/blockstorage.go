package controller

import (
	"log/slog"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/storage/block-storages/v1"
	gateway "github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin/handler"
	kubernetes2domain "github.com/eu-sovereign-cloud/ecp/foundation/persistence/adapters/kubernetes2domain"
)

// BlockStorageController is the specialized controller for BlockStorage resources.
// It uses a GenericController as its base and is configured with the specific
// types and handlers for BlockStorage.
type BlockStorageController GenericController[*regional.BlockStorageDomain]

// NewBlockStorageController creates a new controller for BlockStorage resources.
func NewBlockStorageController(
	client client.Client,
	repo gateway.Repo[*regional.BlockStorageDomain],
	plugin plugin.BlockStorage,
	requeueAfter time.Duration,
	logger *slog.Logger,
	maxConditions int,
) BlockStorageController {
	h := handler.NewBlockStoragePluginHandler(repo, plugin)
	h.MaxConditions = maxConditions

	return (BlockStorageController)(NewGenericController[*regional.BlockStorageDomain](
		client,
		kubernetes2domain.MapCRToBlockStorageDomain,
		h,
		&blockstoragev1.BlockStorage{},
		requeueAfter,
		logger,
		maxConditions,
	))
}
