package cri

import (
	"context"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	runtimev1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// getContainerSpec calls inspect via crictl
func (c *Cri) getContainerSpec(pod *corev1.Pod, ps *runtimev1.PodSandboxStatusResponse) (*runtimev1.ContainerStatusResponse, error) {
	filter := &runtimev1.ContainerStatsFilter{
		PodSandboxId: ps.GetStatus().GetId(),
	}
	stats, err := c.runtimeService.ListContainerStats(context.TODO(), filter)
	if err != nil {
		return nil, err
	}

	// There should be at least one stat from the filter
	if len(stats) < 1 {
		return nil, errors.New(fmt.Sprintf("container stats returned should be 1, but got %d - might be terminated/completed (phase: %s)", len(stats), pod.Status.Phase))
	}

	// Always use stats for first container
	cs, err := c.runtimeService.ContainerStatus(context.TODO(), stats[0].GetAttributes().GetId(), true)
	if err != nil {
		return nil, err
	}

	return cs, nil
}
