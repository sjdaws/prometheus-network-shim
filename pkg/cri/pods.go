package cri

import (
	"context"
	"errors"
	"fmt"

	"github.com/sjdaws/prometheus-network-shim/pkg/cri/runtimes"
	"github.com/sjdaws/prometheus-network-shim/pkg/network"
	corev1 "k8s.io/api/core/v1"
	runtimev1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

// GetPodSpec calls inspectp via crictl
func (c *Cri) GetPodSpec(pod *corev1.Pod) (*runtimes.PodSpec, error) {
	filter := &runtimev1.PodSandboxStatsFilter{
		LabelSelector: map[string]string{"io.kubernetes.pod.uid": string(pod.GetUID())},
	}
	stats, err := c.runtimeService.ListPodSandboxStats(context.TODO(), filter)
	if err != nil {
		return &runtimes.PodSpec{}, err
	}

	// There should be exactly one stat from the filter
	if len(stats) != 1 {
		message := fmt.Sprintf("pod stats returned should be 1, but got %d", len(stats))
		if len(stats) == 0 {
			message = fmt.Sprintf("%s - might be terminated/completed (phase: %s)", message, pod.Status.Phase)
		}
		return &runtimes.PodSpec{}, errors.New(message)
	}

	ps, err := c.runtimeService.PodSandboxStatus(context.TODO(), stats[0].GetAttributes().GetId(), true)
	if err != nil {
		return &runtimes.PodSpec{}, err
	}

	// Get container information
	cs, err := c.getContainerSpec(pod, ps)
	if err != nil {
		return &runtimes.PodSpec{}, err
	}

	podSpec, err := runtimes.Parse(cs, pod.Spec.NodeName, ps)
	if err != nil {
		return &runtimes.PodSpec{}, err
	}

	// Resolve interfaces
	podSpec.Interfaces, _ = network.GetInterfaces(podSpec.NSPath, podSpec.HostNetwork)

	return podSpec, nil
}
