package testing

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewNode returns a new instance of the Node object with the given node.
func NewNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// NodeWithLabels sets the labels of the object.
func NodeWithLabels(obj *corev1.Node, labels map[string]string) {
	if obj.Labels == nil {
		obj.Labels = map[string]string{}
	}

	for k, v := range labels {
		obj.Labels[k] = v
	}
}
