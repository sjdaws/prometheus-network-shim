package cri

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kubernetes-sigs/cri-tools/pkg/common"
)

// getConfigRuntimeEndpoint returns the path to the runtime endpoint
func getConfigRuntimeEndpoint(filename string) (string, error) {
	config, err := common.ReadConfig(fmt.Sprintf("/rootfs%s", filename))
	if err != nil {
		return "", err
	}

	endpoint := config.RuntimeEndpoint
	if endpoint == "" {
		return "", errors.New(fmt.Sprintf("the crictl config file located at %s does not container a runtime-endpoint", filename))
	}

	return strings.Replace(config.RuntimeEndpoint, "://", ":///rootfs", 1), nil
}

// getConfigTimeout returns the timeout
func getConfigTimeout(filename string) int {
	config, err := common.ReadConfig(fmt.Sprintf("/rootfs%s", filename))
	if err != nil {
		// If file doesn't exist, or there is another error, use 0 as timeout
		return 0
	}

	return config.Timeout
}
