package cri

import (
	"context"
	"errors"
	"fmt"

	runtimev1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// getContainerSpec calls inspect via crictl
func (c *Cri) getContainerSpec(podSandboxID string) (*runtimev1.ContainerStatusResponse, error) {
	filter := &runtimev1.ContainerStatsFilter{
		PodSandboxId: podSandboxID,
	}
	stats, err := c.runtimeService.ListContainerStats(context.TODO(), filter)
	if err != nil {
		return nil, err
	}

	// There should be exactly one stat from the filter
	if len(stats) != 1 {
		return nil, errors.New(fmt.Sprintf("container stats returned for %s should be 1, but got %d", podSandboxID, len(stats)))
	}

	cs, err := c.runtimeService.ContainerStatus(context.TODO(), stats[0].GetAttributes().GetId(), true)
	if err != nil {
		return nil, err
	}

	return cs, nil
}
