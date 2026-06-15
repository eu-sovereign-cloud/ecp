package controller

import (
	"log/slog"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/models/regional"
	imagev1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/storage/images/v1"
	gateway "github.com/eu-sovereign-cloud/ecp/foundation/persistence/port"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin/handler"
	kubernetes2domain "github.com/eu-sovereign-cloud/ecp/foundation/persistence/adapters/kubernetes2domain"
)

// ImageController is the specialized controller for Image resources.
// It uses a GenericController as its base and is configured with the specific
// types and handlers for Image.
type ImageController GenericController[*regional.ImageDomain]

// NewImageController creates a new controller for Image resources.
func NewImageController(
	client client.Client,
	repo gateway.Repo[*regional.ImageDomain],
	plugin plugin.Image,
	requeueAfter time.Duration,
	logger *slog.Logger,
	maxConditions int,
) ImageController {
	h := handler.NewImagePluginHandler(repo, plugin)
	h.MaxConditions = maxConditions

	return (ImageController)(NewGenericController[*regional.ImageDomain](
		client,
		kubernetes2domain.MapCRToImageDomain,
		h,
		&imagev1.Image{},
		requeueAfter,
		logger,
		maxConditions,
	))
}
