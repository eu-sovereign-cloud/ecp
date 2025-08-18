package kubeclient

import (
	"fmt"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type KubeClient struct {
	Client dynamic.Interface
}

// New loads kubeconfig and creates a KubeClient.
func New() (*KubeClient, error) {
	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}
	return NewFromConfig(config)
}

// NewFromConfig creates a KubeClient using the provided rest.Config.
func NewFromConfig(cfg *rest.Config) (*KubeClient, error) {
	c := &KubeClient{}
	if c.Client == nil {
		client, err := dynamic.NewForConfig(cfg)
		if err != nil {
			return nil, err
		}
		c.Client = client
	}
	return c, nil
}
