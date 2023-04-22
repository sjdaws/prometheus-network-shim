package metrics

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sjdaws/prometheus-network-shim/pkg/cri/runtimes"
	"k8s.io/klog"
)

// podKey is the key used for the cache
type podKey struct {
	name      string
	namespace string
}

// NetAttachDefPerPod represent the interface attachment definitions bound to a given pod
var NetAttachDefPerPod = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{Name: "pod_interface_shim", Help: "Shim to identify network interfaces assigned to pods."},
	[]string{"container", "id", "image", "interface", "name", "namespace", "node", "pod"},
)
var mtx sync.Mutex
var podSpecs = make(map[podKey]*runtimes.PodSpec)

// DeleteAllForPod stop publishing all the network metrics related to the given pod
func DeleteAllForPod(name string, namespace string) {
	mtx.Lock()
	defer mtx.Unlock()
	podSpec, ok := podSpecs[podKey{name, namespace}]
	if !ok {
		return
	}

	delete(podSpecs, podKey{name, namespace})

	for _, podInterface := range podSpec.Interfaces {
		NetAttachDefPerPod.Delete(createPodSpecLabels(podInterface, podSpec))
	}
}

// Serve serves the network metrics to the given address
func Serve(metricsAddress string, stopCh <-chan struct{}) {
	// Including these stats kills performance when Prometheus polls with multiple targets
	prometheus.Unregister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	prometheus.Unregister(prometheus.NewGoCollector())

	prometheus.MustRegister(NetAttachDefPerPod)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(http.StatusText(http.StatusOK)))
		if err != nil {
			klog.Fatalf("Error writing health check response: %v", err)
		}
	})

	klog.Info("Serving network metrics")
	server := &http.Server{Addr: metricsAddress, Handler: mux}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			klog.Fatalf("Failed serving network metrics: %v", err)
		}
	}()

	go func() {
		<-stopCh
		klog.Info("Received stop signal, closing the network metrics endpoint")
		_ = server.Close()
	}()
}

// UpdateForPod adds metrics for all the provided networks to the given pod
func UpdateForPod(podSpec *runtimes.PodSpec) {
	for _, podInterface := range podSpec.Interfaces {
		// Ignore empty interfaces
		if podInterface == "" {
			continue
		}

		NetAttachDefPerPod.With(createPodSpecLabels(podInterface, podSpec)).Add(0)
	}

	mtx.Lock()
	defer mtx.Unlock()
	podSpecs[podKey{podSpec.Pod, podSpec.Namespace}] = podSpec
}

// createPodSpecLabels create the labels for prometheus from podSpec
func createPodSpecLabels(podInterface string, podSpec *runtimes.PodSpec) prometheus.Labels {
	return prometheus.Labels{
		"container": podSpec.Container,
		"id":        podSpec.ID,
		"image":     podSpec.ImageName,
		"interface": podInterface,
		"name":      podSpec.Name,
		"namespace": podSpec.Namespace,
		"node":      podSpec.NodeName,
		"pod":       podSpec.Pod,
	}
}
