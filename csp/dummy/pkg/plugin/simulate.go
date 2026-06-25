package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	radom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role-assignment/v1"
	roleassignmentconv "github.com/eu-sovereign-cloud/ecp/resource/authorization/role-assignment/v1/backend/kubernetes"
	roledom "github.com/eu-sovereign-cloud/ecp/resource/authorization/role/v1"
	roleconv "github.com/eu-sovereign-cloud/ecp/resource/authorization/role/v1/backend/kubernetes"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/network/v1"
	networkconv "github.com/eu-sovereign-cloud/ecp/resource/network/network/v1/backend/kubernetes"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/block-storage/v1"
	storageconv "github.com/eu-sovereign-cloud/ecp/resource/storage/block-storage/v1/backend/kubernetes"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/image/v1"
	imageconv "github.com/eu-sovereign-cloud/ecp/resource/storage/image/v1/backend/kubernetes"
	wsdom "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1"
	workspaceconv "github.com/eu-sovereign-cloud/ecp/resource/workspace/v1/backend/kubernetes"
)

// simulate reports a long-running operation as still in progress until its
// simulated delay has elapsed, without blocking the reconciliation worker.
// persist is called exactly once, on the first reconciliation, to stamp the
// expiration annotation onto the backing store.
func simulate(
	ctx context.Context,
	op string,
	annotations *map[string]string,
	name string,
	delay time.Duration,
	logger *slog.Logger,
	persist func(context.Context) error,
) error {
	if _, exists := (*annotations)[op]; !exists {
		if *annotations == nil {
			*annotations = make(map[string]string)
		}
		(*annotations)[op] = time.Now().Add(delay).Format(time.RFC3339)

		if err := persist(ctx); err != nil {
			return err
		}
		logger.Info("dummy plugin: stamped expiration annotation", "op", op, "resource_name", name)
	}

	expiration, _ := time.Parse(time.RFC3339, (*annotations)[op])

	if time.Now().Before(expiration) {
		logger.Info("dummy plugin: still processing", "op", op, "resource_name", name)
		return backendport.ErrStillProcessing
	}

	logger.Info("dummy plugin: finished", "op", op, "resource_name", name)
	return nil
}

func newDynamicClient() (*dynamic.DynamicClient, error) {
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restConfig, err := kubeconfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}
	restConfig.QPS = 100
	restConfig.Burst = 200

	return dynamic.NewForConfig(restConfig)
}

func simulateBS(ctx context.Context, op string, resource *bsdom.BlockStorage, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := newDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				storageconv.BlockStorageGVR,
				logger,
				storageconv.BlockStorageToCR,
				storageconv.BlockStorageFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}

func simulateImage(ctx context.Context, op string, resource *imgdom.Image, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := newDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				imageconv.ImageGVR,
				logger,
				imageconv.ImageToCR,
				imageconv.ImageFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}

func simulateRA(ctx context.Context, op string, resource *radom.RoleAssignment, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := newDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				roleassignmentconv.RoleAssignmentGVR,
				logger,
				roleassignmentconv.RoleAssignmentToCR,
				roleassignmentconv.RoleAssignmentFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}

func simulateWS(ctx context.Context, op string, resource *wsdom.Workspace, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := newDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				workspaceconv.WorkspaceGVR,
				logger,
				workspaceconv.WorkspaceToCR,
				workspaceconv.WorkspaceFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}

func simulateNet(ctx context.Context, op string, resource *netdom.Network, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := newDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				networkconv.NetworkGVR,
				logger,
				networkconv.NetworkToCR,
				networkconv.NetworkFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}

func simulateRole(ctx context.Context, op string, resource *roledom.Role, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := newDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				roleconv.RoleGVR,
				logger,
				roleconv.RoleToCR,
				roleconv.RoleFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}
