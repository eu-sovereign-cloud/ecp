package controller

import (
	"log/slog"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/adapter/kubernetes"
	"github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/model/regional"
	gateway "github.com/eu-sovereign-cloud/ecp/foundation/gateway/pkg/port"
	networksv1 "github.com/eu-sovereign-cloud/ecp/foundation/persistence/api/regional/network/networks/v1"

	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin"
	"github.com/eu-sovereign-cloud/ecp/foundation/delegator/pkg/plugin/handler"
)

// NetworkController is the specialized controller for Network resources.
// It uses a GenericController as its base and is configured with the specific
// types and handlers for Network.
type NetworkController GenericController[*regional.NetworkDomain]

// NewNetworkController creates a new controller for Network resources.
func NewNetworkController(
	client client.Client,
	repo gateway.Repo[*regional.NetworkDomain],
	plugin plugin.Network,
	requeueAfter time.Duration,
	logger *slog.Logger,
	maxConditions int,
) NetworkController {
	h := handler.NewNetworkPluginHandler(repo, plugin)
	h.MaxConditions = maxConditions

	return (NetworkController)(NewGenericController[*regional.NetworkDomain](
		client,
		kubernetes.MapCRToNetworkDomain,
		h,
		&networksv1.Network{},
		requeueAfter,
		logger,
		maxConditions,
	))
}
