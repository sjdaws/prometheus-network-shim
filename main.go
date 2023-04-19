package main

import (
	"fmt"

	"github.com/openshift/network-metrics-daemon/pkg/signals"
	"github.com/sjdaws/prometheus-network-shim/pkg/controller"
	"github.com/sjdaws/prometheus-network-shim/pkg/core"
	"github.com/sjdaws/prometheus-network-shim/pkg/cri"
	"github.com/sjdaws/prometheus-network-shim/pkg/metrics"
	"k8s.io/klog"
)

// build is the git version of this program. It is set using build flags in the makefile.
var build = "develop"

func main() {
	klog.InitFlags(nil)
	klog.Info("Version:", build)

	config, err := parseFlags()
	if err != nil {
		klog.Fatalf("Unable to parse flags: %v", err)
	}
	klog.Info(fmt.Sprintf("Starting with config: %+v", config))

	// Create clients
	api, err := core.New(config.kubeConfig, config.masterURL)
	if err != nil {
		klog.Fatalf("Unable to create core api client: %v", err)
	}
	crictl, err := cri.New(config.crictlConfig, config.runtimeEndpoint)
	if err != nil {
		klog.Fatalf("Unable to create crictl client: %v", err)
	}

	// Set up signals, so we handle the first shutdown signal gracefully
	signalHandler := signals.SetupSignalHandler()
	informer := api.SharedIndexInformer(config.nodeName)
	ctrl, err := controller.New(crictl, config.nodeName, api.GetClientset(), informer)
	if err != nil {
		klog.Fatalf("Unable to create controller: %v", err)
	}

	go informer.Run(signalHandler)

	metrics.Serve(config.metricsAddress, signalHandler)

	if err = ctrl.Run(2, signalHandler); err != nil {
		klog.Fatalf("Error running controller: %v", err)
	}
}
