package secapi

import (
	"fmt"

	region "github.com/eu-sovereign-cloud/go-sdk/pkg/spec/foundation.region.v1"
)

type RegionalClient struct {
	authToken string

	ComputeV1   *ComputeV1
	NetworkV1   *NetworkV1
	StorageV1   *StorageV1
	WorkspaceV1 *WorkspaceV1
}

func newRegionalClient(authToken string, region *region.Region) (*RegionalClient, error) {
	client := &RegionalClient{
		authToken: authToken,
	}

	// Initializes computeV1 API client
	if err := initRegionalAPI(client, "seca.compute", region, newComputeV1, client.setComputeV1); err != nil {
		return nil, err
	}

	// Initializes networkV1 API client
	if err := initRegionalAPI(client, "seca.network", region, newNetworkV1, client.setNetworkV1); err != nil {
		return nil, err
	}

	// Initializes storageV1 API client
	if err := initRegionalAPI(client, "seca.storage", region, newStorageV1, client.setStorageV1); err != nil {
		return nil, err
	}

	// Initializes workspaceV1 API client
	if err := initRegionalAPI(client, "seca.workspace", region, newWorkspaceV1, client.setWorkspaceV1); err != nil {
		return nil, err
	}

	return client, nil
}

func initRegionalAPI[T any](client *RegionalClient, name string, region *region.Region, newFunc func(client *RegionalClient, url string) (*T, error), setFunc func(*T)) error {
	provider, err := findRegionalProvider(name, region)
	if err != nil {
		return err
	}

	api, err := newFunc(client, provider.Url)
	if err != nil {
		return err
	}

	setFunc(api)
	return nil
}

func findRegionalProvider(name string, region *region.Region) (*region.Provider, error) {
	for _, provider := range region.Spec.Providers {
		if provider.Name == name {
			return &provider, nil
		}
	}

	return nil, fmt.Errorf("provider %s not found in region", name)
}

func (client *RegionalClient) setComputeV1(compute *ComputeV1) {
	client.ComputeV1 = compute
}

func (client *RegionalClient) setNetworkV1(network *NetworkV1) {
	client.NetworkV1 = network
}

func (client *RegionalClient) setStorageV1(storage *StorageV1) {
	client.StorageV1 = storage
}

func (client *RegionalClient) setWorkspaceV1(workspace *WorkspaceV1) {
	client.WorkspaceV1 = workspace
}
