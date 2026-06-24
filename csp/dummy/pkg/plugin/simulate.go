package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	storageconv "github.com/eu-sovereign-cloud/ecp/resource/storage/block-storage/v1/backend/kubernetes"
	workspaceconv "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"

	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"

	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/block-storage/v1"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
)

// simulate reports a long-running operation as still in progress until its
// simulated delay has elapsed, without blocking the reconciliation worker.
func simulateBS(ctx context.Context, op string, resource *bsdom.BlockStorage, delay time.Duration, logger *slog.Logger) error {
	if _, exists := resource.Annotations[op]; !exists {
		if resource.Annotations == nil {
			resource.Annotations = make(map[string]string)
		}
		resource.Annotations[op] = time.Now().Add(delay).Format(time.RFC3339)

		kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{},
		)
		restConfig, err := kubeconfig.ClientConfig()
		if err != nil {
			return fmt.Errorf("failed to get kubeconfig: %w", err)
		}
		restConfig.QPS = 100
		restConfig.Burst = 200

		// Initialize dynamic client
		dynamicClient, err := dynamic.NewForConfig(restConfig)
		if err != nil {
			return fmt.Errorf("failed to create dynamic client: %w", err)
		}

		blockStorageRepo := kubernetesadapter.NewRepoAdapter(
			dynamicClient,
			storageconv.BlockStorageGVR,
			logger,
			storageconv.MapBlockStorageDomainToCR,
			storageconv.MapCRToBlockStorageDomain,
		)
		_, err = blockStorageRepo.Update(ctx, resource)
		if err != nil {
			return err
		}
		logger.Info("Updated resource annotations for operation %s on block storage %s", op, resource.GetName())
	}

	expiration, _ := time.Parse(time.RFC3339, resource.Annotations[op])

	if time.Now().Before(expiration) {
		logger.Info("dummy plugin: still processing",
			"op", op, "resource_name", resource.GetName())

		return backendport.ErrStillProcessing
	}

	logger.Info("dummy plugin: finished",
		"op", op, "resource_name", resource.GetName())

	return nil
}

func simulateWS(ctx context.Context, op string, resource *wsdom.Workspace, delay time.Duration, logger *slog.Logger) error {
	if _, exists := resource.Annotations[op]; !exists {
		if resource.Annotations == nil {
			resource.Annotations = make(map[string]string)
		}
		resource.Annotations[op] = time.Now().Add(delay).Format(time.RFC3339)

		kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			clientcmd.NewDefaultClientConfigLoadingRules(),
			&clientcmd.ConfigOverrides{},
		)
		restConfig, err := kubeconfig.ClientConfig()
		if err != nil {
			return fmt.Errorf("failed to get kubeconfig: %w", err)
		}
		restConfig.QPS = 100
		restConfig.Burst = 200

		// Initialize dynamic client
		dynamicClient, err := dynamic.NewForConfig(restConfig)
		if err != nil {
			return fmt.Errorf("failed to create dynamic client: %w", err)
		}

		workspaceRepo := kubernetesadapter.NewRepoAdapter(
			dynamicClient,
			workspaceconv.WorkspaceGVR,
			logger,
			workspaceconv.MapWorkspaceDomainToCR,
			workspaceconv.MapCRToWorkspaceDomain,
		)
		_, err = workspaceRepo.Update(ctx, resource)
		if err != nil {
			return err
		}
		logger.Info("Updated resource annotations for operation %s on workspace %s", op, resource.GetName())
	}

	expiration, _ := time.Parse(time.RFC3339, resource.Annotations[op])

	if time.Now().Before(expiration) {
		logger.Info("dummy plugin: still processing",
			"op", op, "resource_name", resource.GetName())

		return backendport.ErrStillProcessing
	}

	logger.Info("dummy plugin: finished",
		"op", op, "resource_name", resource.GetName())

	return nil
}
