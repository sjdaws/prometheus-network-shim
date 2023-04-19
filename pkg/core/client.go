package core

import (
	"errors"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	clientset *kubernetes.Clientset
}

// New creates a new kubernetes client
func New(kubeConfig string, masterURL string) (*Client, error) {
	config, err := clientcmd.BuildConfigFromFlags(masterURL, kubeConfig)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error building kubernetes rest config: %v", err))
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error creating kubernetes client: %v", err))
	}

	return &Client{
		clientset: clientset,
	}, nil
}

// GetClientset returns the underlying kubernetes clientset
func (c *Client) GetClientset() *kubernetes.Clientset {
	return c.clientset
}
