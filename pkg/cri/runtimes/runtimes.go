package runtimes

import (
	"errors"
	"fmt"
	"reflect"

	runtimev1 "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type ContainerSpec struct {
	ID    string
	Name  string
	Image string
}

type PodSpec struct {
	Container   string
	HostNetwork bool
	ID          string
	ImageName   string
	Interfaces  []string
	Name        string
	Namespace   string
	NodeName    string
	NSPath      string
	Parser      string
	Pod         string
	Runtime     string
	SandboxID   string
	UID         string
}

type parser interface {
	parse(cs *runtimev1.ContainerStatusResponse, nodeName string, ps *runtimev1.PodSandboxStatusResponse) runtime
}

type runtime interface {
	getPodSpec() *PodSpec
	success() bool
}

// Parse parses a response and returns a podspec
func Parse(cs *runtimev1.ContainerStatusResponse, nodeName string, ps *runtimev1.PodSandboxStatusResponse) (*PodSpec, error) {
	// Any runtimes base structs must be added here to be picked up
	parsers := []parser{
		&crioParser{},
		&genericLinuxParser{},
	}

	for _, test := range parsers {
		result := attempt(test, cs, nodeName, ps)

		if result.success() {
			podSpec := result.getPodSpec()
			podSpec.Container = cs.GetStatus().GetMetadata().GetName()
			podSpec.ImageName = cs.GetStatus().GetImage().GetImage()
			podSpec.NodeName = nodeName
			podSpec.Parser = reflect.TypeOf(test).String()

			return podSpec, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("unsure how to handle stat %s", ps.String()))
}

// attempt attempts to parse stat using labels relevant to that runtime
func attempt(test parser, cs *runtimev1.ContainerStatusResponse, nodeName string, ps *runtimev1.PodSandboxStatusResponse) runtime {
	return test.parse(cs, nodeName, ps)
}
