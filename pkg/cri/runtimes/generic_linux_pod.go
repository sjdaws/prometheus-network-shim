package runtimes

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	runtimev1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type genericLinux struct {
	*PodSpec
}

type genericLinuxParser struct {
}

type GenericLinuxPodSpec struct {
	RuntimeSpec GenericLinuxRuntimeSpec `json:"runtimeSpec"`
}

type GenericLinuxRuntimeSpec struct {
	Annotations GenericLinuxAnnotations `json:"annotations"`
	Hostname    string                  `json:"hostname"`
	Linux       GenericLinuxLinux       `json:"linux"`
}

type GenericLinuxAnnotations struct {
	Manager string `json:"io.container.manager"`
}

type GenericLinuxLinux struct {
	CgroupsPath string                  `json:"cgroupsPath"`
	Namespaces  []GenericLinuxNamespace `json:"namespaces"`
}

type GenericLinuxNamespace struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// getNSPath gets the nsPath from somewhere
func (g *genericLinuxParser) getNSPath(genericPS GenericLinuxPodSpec) (string, error) {
	for _, namespace := range genericPS.RuntimeSpec.Linux.Namespaces {
		if namespace.Type == "network" {
			return namespace.Path, nil
		}
	}

	return "", errors.New("unable to find nsPath in pod status")
}

// getPodSpec returns the parsed pod spec
func (g *genericLinux) getPodSpec() *PodSpec {
	return g.PodSpec
}

// parse attempts to parse ps using crio labels
func (g *genericLinuxParser) parse(cs *runtimev1.ContainerStatusResponse, nodeName string, ps *runtimev1.PodSandboxStatusResponse) runtime {
	var genericPS GenericLinuxPodSpec
	err := json.Unmarshal([]byte(ps.GetInfo()["info"]), &genericPS)
	if err != nil {
		return &genericLinux{}
	}

	podSpecID := []string{"/kubepods.slice"}
	cgroupsPathParts := strings.Split(genericPS.RuntimeSpec.Linux.CgroupsPath, ":")
	if len(cgroupsPathParts) == 3 {
		containerParts := strings.Split(cgroupsPathParts[0], "-")
		if len(containerParts) == 3 {
			podSpecID = append(podSpecID, fmt.Sprintf("kubepods-%s.slice", containerParts[1]))
		}
		podSpecID = append(podSpecID, cgroupsPathParts[0], fmt.Sprintf("%s-%s.scope", cgroupsPathParts[1], cs.GetStatus().GetId()))
	}

	nsPath, err := g.getNSPath(genericPS)
	if err != nil {
		return &crio{}
	}

	labels := ps.GetStatus().GetLabels()
	return &genericLinux{
		&PodSpec{
			HostNetwork: strings.EqualFold(genericPS.RuntimeSpec.Hostname, nodeName),
			ID:          strings.Join(podSpecID, "/"),
			Name:        fmt.Sprintf("k8s_%s_%s_%s_%s_1", cs.GetStatus().GetMetadata().GetName(), labels["io.kubernetes.pod.name"], labels["io.kubernetes.pod.namespace"], ps.GetStatus().GetMetadata().GetUid()),
			Namespace:   labels["io.kubernetes.pod.namespace"],
			NSPath:      nsPath,
			Pod:         labels["io.kubernetes.pod.name"],
			Runtime:     genericPS.RuntimeSpec.Annotations.Manager,
			SandboxID:   ps.GetStatus().GetId(),
			UID:         ps.GetStatus().GetMetadata().GetUid(),
		},
	}
}

// success determines if the runtime is generic
func (g *genericLinux) success() bool {
	return g.Runtime != ""
}
