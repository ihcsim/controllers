package controller

import (
	crdv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/clientset/versioned"
	"k8s.io/client-go/kubernetes"
)

// Controller knows how to reconcile kubelet upgrades to match the in-cluster
// states with the desired states.
type Controller struct {
	clientsetsK8s kubernetes.Interface
	clientsetsCRD crdv1alpha1.Interface
}

// New returns a new instance of the controller.
func New(clientsetsK8s kubernetes.Interface, clientsetsCRD crdv1alpha1.Interface) *Controller {
	return &Controller{
		clientsetsK8s: clientsetsK8s,
		clientsetsCRD: clientsetsCRD,
	}
}

// Sync will set up the event handlers for our custom resources, as well as
// syncing informer caches and starting workers. It will block until stopCh is
// closed, at which point it will shutdown the workqueue and wait for
func (c *Controller) Sync(stopCh <-chan struct{}) error {

	return nil
}
