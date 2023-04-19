package core

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) ListNodePods(nodeName string) (*corev1.PodList, error) {
	options := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	}

	return c.clientset.CoreV1().Pods(metav1.NamespaceAll).List(context.Background(), options)
}
