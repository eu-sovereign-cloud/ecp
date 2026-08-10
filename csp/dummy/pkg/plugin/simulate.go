package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	kubernetesadapter "github.com/eu-sovereign-cloud/ecp/framework/backend/kubernetes"
	backendport "github.com/eu-sovereign-cloud/ecp/framework/kernel/port/backend"
	instancedom "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance"
	instanceconv "github.com/eu-sovereign-cloud/ecp/resource/compute/v1/instance/backend/kubernetes"
	internetgatewaydom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway"
	internetgatewayconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/internet-gateway/backend/kubernetes"
	netdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network"
	networkconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/network/backend/kubernetes"
	nicdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic"
	nicconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/nic/backend/kubernetes"
	publicipdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip"
	publicipconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/public-ip/backend/kubernetes"
	routetabledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table"
	routetableconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/route-table/backend/kubernetes"
	securitygroupdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group"
	securitygroupruledom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule"
	securitygroupruleconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group-rule/backend/kubernetes"
	securitygroupconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/security-group/backend/kubernetes"
	subnetdom "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet"
	subnetconv "github.com/eu-sovereign-cloud/ecp/resource/network/v1/subnet/backend/kubernetes"
	bsdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage"
	storageconv "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/block-storage/backend/kubernetes"
	imgdom "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image"
	imageconv "github.com/eu-sovereign-cloud/ecp/resource/storage/v1/image/backend/kubernetes"
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

// sharedDynamicClient returns the one client every persist path here shares.
//
// Building it per call - which is what each simulate* and applyUpdate used to do - re-read the
// kubeconfig from disk and re-negotiated TLS on the reconcile path, and handed back a fresh rate
// limiter each time, so the QPS and burst configured below applied to a population of one request
// and bounded nothing. The client is safe for concurrent use and the kubeconfig cannot change under
// a running plugin, so once is enough. A failure is cached with it: if the kubeconfig cannot be
// loaded at all, every later attempt would fail the same way.
var sharedDynamicClient = sync.OnceValues(func() (*dynamic.DynamicClient, error) {
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
})

func simulateBS(ctx context.Context, op string, resource *bsdom.BlockStorage, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := sharedDynamicClient()
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
			dynamicClient, err := sharedDynamicClient()
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

func simulateWS(ctx context.Context, op string, resource *wsdom.Workspace, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := sharedDynamicClient()
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

func simulateInstance(ctx context.Context, op string, resource *instancedom.Instance, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := sharedDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				instanceconv.InstanceGVR,
				logger,
				instanceconv.InstanceToCR,
				instanceconv.InstanceFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}

func simulateNic(ctx context.Context, op string, resource *nicdom.Nic, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := sharedDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				nicconv.NICGVR,
				logger,
				nicconv.NicToCR,
				nicconv.NicFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}

func simulatePublicIp(ctx context.Context, op string, resource *publicipdom.PublicIp, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := sharedDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				publicipconv.PublicIPGVR,
				logger,
				publicipconv.PublicIpToCR,
				publicipconv.PublicIpFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}

func simulateInternetGateway(ctx context.Context, op string, resource *internetgatewaydom.InternetGateway, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := sharedDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				internetgatewayconv.InternetGatewayGVR,
				logger,
				internetgatewayconv.InternetGatewayToCR,
				internetgatewayconv.InternetGatewayFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}

func simulateNet(ctx context.Context, op string, resource *netdom.Network, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := sharedDynamicClient()
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

func simulateRouteTable(ctx context.Context, op string, resource *routetabledom.RouteTable, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := sharedDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				routetableconv.RouteTableGVR,
				logger,
				routetableconv.RouteTableToCR,
				routetableconv.RouteTableFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}

func simulateSubnet(ctx context.Context, op string, resource *subnetdom.Subnet, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := sharedDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				subnetconv.SubnetGVR,
				logger,
				subnetconv.SubnetToCR,
				subnetconv.SubnetFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}

func simulateSecurityGroup(ctx context.Context, op string, resource *securitygroupdom.SecurityGroup, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := sharedDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				securitygroupconv.SecurityGroupGVR,
				logger,
				securitygroupconv.SecurityGroupToCR,
				securitygroupconv.SecurityGroupFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}

func simulateSecurityGroupRule(ctx context.Context, op string, resource *securitygroupruledom.SecurityGroupRule, delay time.Duration, logger *slog.Logger) error {
	return simulate(ctx, op, &resource.Annotations, resource.GetName(), delay, logger,
		func(ctx context.Context) error {
			dynamicClient, err := sharedDynamicClient()
			if err != nil {
				return err
			}
			repo := kubernetesadapter.NewRepoAdapter(
				dynamicClient,
				securitygroupruleconv.SecurityGroupRuleGVR,
				logger,
				securitygroupruleconv.SecurityGroupRuleToCR,
				securitygroupruleconv.SecurityGroupRuleFromCR,
			)
			_, err = repo.Update(ctx, resource)
			return err
		},
	)
}
