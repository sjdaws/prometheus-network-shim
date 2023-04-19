package network

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/utils/exec"
	"k8s.io/utils/nsenter"
)

// GetInterfaces returns pod interfaces
func GetInterfaces(nsPath string, hostNetwork bool) ([]string, error) {
	if hostNetwork {
		return getHostInterfaces()
	}

	return getNSPathInterfaces(nsPath)
}

// getHostInterfaces returns the host interfaces
func getHostInterfaces() ([]string, error) {
	hostInterfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	// Get the lowest interface that isn't lo
	for _, hostInterface := range hostInterfaces {
		if !strings.EqualFold(hostInterface.Name, "lo") {
			return []string{hostInterface.Name}, nil
		}
	}

	return nil, errors.New("no host interfaces found")
}

// getNSPathInterfaces returns the interfaces found in an nspath file
//
// This is hacky, taken from https://platform9.com/kb/kubernetes/how-to-identify-the-virtual-interface-of-a-pod-in-the-root-name
func getNSPathInterfaces(nsPath string) ([]string, error) {
	ns, err := nsenter.NewNsenter("/rootfs", exec.New())
	if err != nil {
		return nil, err
	}

	cmd := ns.Command("nsenter", fmt.Sprintf("--net=%s", nsPath), "ethtool", "-S", "eth0")
	result, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var interfaces []string
	lines := strings.Split(string(result), "\n")
	for _, line := range lines {
		// Look for the peer_ifindex line
		if !strings.Contains(line, "peer") {
			continue
		}

		// Grab the peer id
		regex := regexp.MustCompile("[0-9]+")
		matches := regex.FindAllString(line, -1)
		for _, match := range matches {
			interfaceID, _ := strconv.Atoi(match)
			podInterface, err := net.InterfaceByIndex(interfaceID)
			if err != nil {
				continue
			}

			interfaces = append(interfaces, podInterface.Name)
		}
	}

	if len(interfaces) < 1 {
		return nil, errors.New(fmt.Sprintf("no interfaces found in %s", nsPath))
	}

	return interfaces, nil
}
