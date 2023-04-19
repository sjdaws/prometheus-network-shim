package cri

import (
	"context"
	"errors"
	"fmt"

	"github.com/sjdaws/prometheus-network-shim/pkg/cri/runtimes"
	"github.com/sjdaws/prometheus-network-shim/pkg/network"
	"k8s.io/apimachinery/pkg/types"
	runtimev1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// GetPodSpec calls inspectp via crictl
func (c *Cri) GetPodSpec(nodeName string, uid types.UID) (*runtimes.PodSpec, error) {
	filter := &runtimev1.PodSandboxStatsFilter{
		LabelSelector: map[string]string{"io.kubernetes.pod.uid": string(uid)},
	}
	stats, err := c.runtimeService.ListPodSandboxStats(context.TODO(), filter)
	if err != nil {
		return &runtimes.PodSpec{}, err
	}

	// There should be exactly one stat from the filter
	if len(stats) != 1 {
		return &runtimes.PodSpec{}, errors.New(fmt.Sprintf("pod stats returned for %s should be 1, but got %d", uid, len(stats)))
	}

	ps, err := c.runtimeService.PodSandboxStatus(context.TODO(), stats[0].GetAttributes().GetId(), true)
	if err != nil {
		return &runtimes.PodSpec{}, err
	}

	// Get container information
	cs, err := c.getContainerSpec(ps.GetStatus().GetId())
	if err != nil {
		return &runtimes.PodSpec{}, err
	}

	podSpec, err := runtimes.Parse(cs, nodeName, ps)
	if err != nil {
		return &runtimes.PodSpec{}, err
	}

	// Resolve interfaces
	podSpec.Interfaces, _ = network.GetInterfaces(podSpec.NSPath, podSpec.HostNetwork)

	return podSpec, nil
}
