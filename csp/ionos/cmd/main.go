package main

import (
	"log/slog"
	"os"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	ionosapis "github.com/ionos-cloud/provider-upjet-ionoscloud/apis/namespaced/compute/v1alpha1"

	"github.com/eu-sovereign-cloud/ecp/csp/ionos/pkg/controllerset"
	frameworkbuilder "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes/builder"
	resourcescheme "github.com/eu-sovereign-cloud/ecp/resource/scheme"
)

func main() {
	d, err := frameworkbuilder.NewDelegator(ctrl.Options{
		Metrics: metricsserver.Options{
			SecureServing: false,
			BindAddress:   ":8083",
		},
	}, resourcescheme.AddToScheme, ionosapis.AddToScheme)
	if err != nil {
		slog.Error("unable to bootstrap delegator", "error", err)
		os.Exit(1)
	}

	controllerOpts := []frameworkbuilder.Option{
		frameworkbuilder.WithLogger(d.Logger.With("component", "controller-set")),
		frameworkbuilder.WithRequeueAfter(1 * time.Second),
		frameworkbuilder.WithMaxConditions(5),
	}
	controllerset.Add(d.Controllers, d.Manager, d.Dynamic, d.Clientset, d.Logger, controllerOpts...)

	if err := d.Run(ctrl.SetupSignalHandler()); err != nil {
		d.Logger.Error("delegator stopped", "error", err)
		os.Exit(1)
	}
}
