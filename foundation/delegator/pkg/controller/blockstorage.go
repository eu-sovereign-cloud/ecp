package controller

import (
	"log/slog"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	blockstoragev1 "github.com/eu-sovereign-cloud/ecp/foundation/api/regional/storage/block-storages/v1"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	gateway "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin/handler"
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
) *BlockStorageController {
	return (*BlockStorageController)(NewGenericController[*regional.BlockStorageDomain](
		client,
		kubernetes.MapCRToBlockStorageDomain,
		handler.NewBlockStoragePluginHandler(repo, plugin),
		&blockstoragev1.BlockStorage{},
		requeueAfter,
		logger,
	))
}
