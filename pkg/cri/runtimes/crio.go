package runtimes

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	runtimev1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type crio struct {
	*PodSpec
}

type crioParser struct {
}

type CrioContainerSpec struct {
	RuntimeSpec CrioContainerRuntimeSpec `json:"runtimeSpec"`
}

type CrioContainerRuntimeSpec struct {
	Annotations CrioContainerAnnotations `json:"annotations"`
}

type CrioContainerAnnotations struct {
	Name string `json:"io.kubernetes.cri-o.Name"`
}

type CrioPodSpec struct {
	RuntimeSpec CrioPodRuntimeSpec `json:"runtimeSpec"`
}

type CrioPodRuntimeSpec struct {
	Annotations CrioPodAnnotations `json:"annotations"`
	Linux       CrioPodLinux       `json:"linux"`
}

type CrioPodAnnotations struct {
	CgroupParent  string `json:"io.kubernetes.cri-o.CgroupParent"`
	CNIResult     string `json:"io.kubernetes.cri-o.CNIResult"`
	ContainerName string `json:"io.kubernetes.cri-o.ContainerName"`
	HostNetwork   string `json:"io.kubernetes.cri-o.HostNetwork"`
	Manager       string `json:"io.container.manager"`
}

type CrioCNIResult struct {
	Interfaces []CrioInterface `json:"interfaces"`
}

type CrioInterface struct {
	Sandbox string `json:"sandbox"`
}

type CrioPodLinux struct {
	Namespaces []CrioPodNamespace `json:"namespaces"`
}

type CrioPodNamespace struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

// getCNINSPath gets the nsPath from CNI label data
func (c *crioParser) getCNINSPath(criPS CrioPodSpec) string {
	if criPS.RuntimeSpec.Annotations.CNIResult == "" {
		return ""
	}

	var cniResult CrioCNIResult
	err := json.Unmarshal([]byte(criPS.RuntimeSpec.Annotations.CNIResult), &cniResult)
	if err != nil {
		return ""
	}

	// If there are no interfaces, skip
	if len(cniResult.Interfaces) < 1 {
		return ""
	}

	return cniResult.Interfaces[0].Sandbox
}

// getLinuxNSPath gets the nsPath from linux data
func (c *crioParser) getLinuxNSPath(criPS CrioPodSpec) string {
	for _, namespace := range criPS.RuntimeSpec.Linux.Namespaces {
		if namespace.Type == "network" {
			return namespace.Path
		}
	}

	return ""
}

// getNSPath gets the nsPath from somewhere
func (c *crioParser) getNSPath(criPS CrioPodSpec) (string, error) {
	nsPath := c.getCNINSPath(criPS)
	if nsPath != "" {
		return nsPath, nil
	}

	nsPath = c.getLinuxNSPath(criPS)
	if nsPath != "" {
		return nsPath, nil
	}

	return "", errors.New("unable to find nsPath in pod status")
}

// getPodSpec returns the parsed pod spec
func (c *crio) getPodSpec() *PodSpec {
	return c.PodSpec
}

// parse attempts to parse ps using crio labels
func (c *crioParser) parse(cs *runtimev1.ContainerStatusResponse, _ string, ps *runtimev1.PodSandboxStatusResponse) runtime {
	var criPS CrioPodSpec
	err := json.Unmarshal([]byte(ps.GetInfo()["info"]), &criPS)
	if err != nil {
		return &crio{}
	}

	var criCS CrioContainerSpec
	err = json.Unmarshal([]byte(cs.GetInfo()["info"]), &criCS)
	if err != nil {
		return &crio{}
	}

	podSpecID := []string{"/kubepods.slice"}
	containerParts := strings.Split(criPS.RuntimeSpec.Annotations.CgroupParent, "-")
	if len(containerParts) == 3 {
		podSpecID = append(podSpecID, fmt.Sprintf("kubepods-%s.slice", containerParts[1]))
	}
	podSpecID = append(podSpecID, criPS.RuntimeSpec.Annotations.CgroupParent, fmt.Sprintf("crio-%s.scope", cs.GetStatus().GetId()))

	nsPath, err := c.getNSPath(criPS)
	if err != nil {
		return &crio{}
	}

	labels := ps.GetStatus().GetLabels()
	return &crio{
		&PodSpec{
			HostNetwork: strings.EqualFold(criPS.RuntimeSpec.Annotations.HostNetwork, "true"),
			ID:          strings.Join(podSpecID, "/"),
			Name:        criCS.RuntimeSpec.Annotations.Name,
			Namespace:   labels["io.kubernetes.pod.namespace"],
			NSPath:      nsPath,
			Pod:         labels["io.kubernetes.pod.name"],
			Runtime:     criPS.RuntimeSpec.Annotations.Manager,
			SandboxID:   ps.GetStatus().GetId(),
			UID:         ps.GetStatus().GetMetadata().GetUid(),
		},
	}
}

// success determines if the runtime is crio
func (c *crio) success() bool {
	return c.Runtime == "cri-o"
}
