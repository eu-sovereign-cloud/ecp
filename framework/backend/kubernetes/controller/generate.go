//go:build !crdgen

package controller

//go:generate mockgen -package controller -destination=zz_mock_plugin_handler_test.go github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend PluginHandler
//go:generate mockgen -package controller -destination=zz_mock_client_test.go sigs.k8s.io/controller-runtime/pkg/client Client
//go:generate mockgen -package controller -destination=zz_mock_status_writer_test.go sigs.k8s.io/controller-runtime/pkg/client StatusWriter
