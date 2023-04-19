package cri

import (
	criApi "k8s.io/cri-api/pkg/apis"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"
)

type Cri struct {
	runtimeService criApi.RuntimeService
}

// New creates a new cri instance.
func New(config string, endpoint string) (*Cri, error) {
	timeout := getTimeout(getConfigTimeout(config))

	var err error
	if endpoint == "" {
		endpoint, err = getConfigRuntimeEndpoint(config)
		if err != nil {
			return nil, err
		}
	}

	runtimeService, err := remote.NewRemoteRuntimeService(endpoint, timeout, nil)
	if err != nil {
		return nil, err
	}

	return &Cri{
		runtimeService: runtimeService,
	}, nil
}
