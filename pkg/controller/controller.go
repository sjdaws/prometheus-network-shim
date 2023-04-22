package controller

import (
	"errors"
	"fmt"
	"time"

	"github.com/sjdaws/prometheus-network-shim/pkg/cri"
	"github.com/sjdaws/prometheus-network-shim/pkg/metrics"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog"
)

// Controller is the controller implementation for Foo resources
type Controller struct {
	crictl        *cri.Cri
	kubeclientset kubernetes.Interface
	podsSynced    cache.InformerSynced
	indexer       cache.Indexer
	workqueue     workqueue.RateLimitingInterface
}

// New returns a new controller listening to pods.
func New(crictl *cri.Cri, nodeName string, kubeclientset kubernetes.Interface, informer cache.SharedIndexInformer) (*Controller, error) {
	controller := &Controller{
		crictl:        crictl,
		kubeclientset: kubeclientset,
		indexer:       informer.GetIndexer(),
		podsSynced:    informer.HasSynced,
		workqueue:     workqueue.NewRateLimitingQueueWithConfig(workqueue.DefaultControllerRateLimiter(), workqueue.RateLimitingQueueConfig{Name: "Pods"}),
	}

	klog.Info("Setting up event handlers")

	_, err := informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			if pod.Spec.NodeName != nodeName {
				return
			}
			controller.enqueuePod(pod)
		},
		UpdateFunc: func(_, obj interface{}) {
			pod := obj.(*corev1.Pod)
			if pod.Spec.NodeName != nodeName {
				return
			}
			controller.enqueuePod(pod)
		},
		DeleteFunc: func(obj interface{}) {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
				if !ok {
					runtime.HandleError(errors.New(fmt.Sprintf("couldn't get object from tombstone: %+v", obj)))
					return
				}
				pod, ok = tombstone.Obj.(*corev1.Pod)
				if !ok {
					runtime.HandleError(errors.New(fmt.Sprintf("tombstone contained object that is not an RC: %#v", obj)))
					return
				}
			}
			if pod.Spec.NodeName != nodeName {
				return
			}
			controller.enqueuePod(pod)
		},
	})
	if err != nil {
		return nil, err
	}

	return controller, nil
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shut down the workqueue and wait for
// workers to finish processing their current work items.
func (c *Controller) Run(threadiness int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting pod controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.podsSynced); !ok {
		return errors.New("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// enqueuePod adds a pod to the workqueue
func (c *Controller) enqueuePod(obj interface{}) {
	var key string
	var err error

	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}

	c.workqueue.Add(key)
}

// podHandler receives a pod and updates the related pod network metrics
func (c *Controller) podHandler(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(errors.New(fmt.Sprintf("invalid resource key: %s", key)))
		return nil
	}
	obj, exists, err := c.indexer.GetByKey(key)
	// Get the Pod resource with this namespace/name
	if err != nil {
		if apierrors.IsNotFound(err) {
			metrics.DeleteAllForPod(name, namespace)
			return nil
		}
		return err
	}

	if !exists {
		metrics.DeleteAllForPod(name, namespace)
		return nil
	}

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		runtime.HandleError(errors.New(fmt.Sprintf("invalid object for key: %s", key)))
		return nil
	}

	klog.Infof("Received pod: %s", pod.Name)

	podSpec, err := c.crictl.GetPodSpec(pod)
	if err != nil {
		return err
	}

	if podSpec.HostNetwork {
		klog.Warningf("Not mapping pod %s to interfaces %v as this pod is using host network", podSpec.Pod, podSpec.Interfaces)
		metrics.DeleteAllForPod(podSpec.Pod, podSpec.Namespace)
		return nil
	}

	// As an interface might have been removed from the pod (or changed)
	// and eventually re-add them, as the chance of having the networks changed is
	// pretty low
	klog.Infof("Mapping pod %s to interfaces %v", podSpec.Pod, podSpec.Interfaces)
	metrics.DeleteAllForPod(podSpec.Pod, podSpec.Namespace)
	metrics.UpdateForPod(podSpec)

	return nil
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *Controller) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func, so we can defer c.workqueue.Done.
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			runtime.HandleError(errors.New(fmt.Sprintf("expected string in workqueue but got %#v", obj)))
			return nil
		}
		if err := c.podHandler(key); err != nil {
			c.workqueue.AddRateLimited(key)

			return errors.New(fmt.Sprintf("error syncing %s: %v, requeuing", key, err))
		}

		c.workqueue.Forget(obj)
		klog.Infof("Successfully synced %s", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}

	return true
}

// runWorker processes the next work item
func (c *Controller) runWorker() {
	for c.processNextWorkItem() {
	}
}
