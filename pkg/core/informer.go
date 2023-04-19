package core

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// SharedIndexInformer returns an informer for a specific node
func (c *Client) SharedIndexInformer(nodeName string) cache.SharedIndexInformer {
	fieldSelector := fmt.Sprintf("spec.nodeName=%s", nodeName)

	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				options.FieldSelector = fieldSelector
				return c.clientset.CoreV1().Pods(metav1.NamespaceAll).List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.FieldSelector = fieldSelector
				return c.clientset.CoreV1().Pods(metav1.NamespaceAll).Watch(context.Background(), options)
			},
		},
		&corev1.Pod{},
		time.Second*30,
		cache.Indexers{},
	)
}
