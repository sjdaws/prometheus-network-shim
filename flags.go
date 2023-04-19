package main

import (
	"errors"
	"flag"
)

// flags represents the flags to parse from the command line
type flags struct {
	crictlConfig    string
	kubeConfig      string
	masterURL       string
	metricsAddress  string
	nodeName        string
	runtimeEndpoint string
}

// parseFlags parses flags from the command line
func parseFlags() (*flags, error) {
	var config flags

	flag.StringVar(&config.crictlConfig, "crictl-config", "/etc/crictl.yaml", "The location of the crictl.yaml file, usually /etc/crictl.yaml. Not used if --runtime-endpoint is set.")
	flag.StringVar(&config.kubeConfig, "kube-config", "", "Path to a kubeconfig.")
	flag.StringVar(&config.masterURL, "master-url", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig.")
	flag.StringVar(&config.metricsAddress, "metrics-listen-address", ":9091", "Metrics server listen address, either address:port or :port.")
	flag.StringVar(&config.nodeName, "node-name", "", "The node the daemon is running on.")
	flag.StringVar(&config.runtimeEndpoint, "runtime-endpoint", "", "The runtime endpoint to use, e.g. unix:///run/containerd/containerd.sock.")
	flag.Parse()

	err := validateFlags(config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// validateFlags performs basic error checking on parsed flags
func validateFlags(config flags) error {
	// Make sure current node is supplied
	if config.nodeName == "" {
		return errors.New("--node-name must be supplied")
	}

	// Make sure runtime endpoint is supplied
	if config.crictlConfig == "" && config.runtimeEndpoint == "" {
		return errors.New("--crictl-config or --runtime-endpoint must be supplied")
	}

	return nil
}
