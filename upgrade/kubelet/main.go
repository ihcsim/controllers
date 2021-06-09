package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/ihcsim/controllers/upgrade/kubelet/pkg/controller"
	crdv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	parseFlags()
	k8s, crd, err := clientsets()
	if err != nil {
		klog.Exit(err)
	}

	stop := handleSignal()
	c := controller.New(k8s, crd)
	if err := c.Sync(stop); err != nil {
		klog.Errorf("controller sync failed: ", err)
	}
}

func handleSignal() chan struct{} {
	var (
		kill = make(chan os.Signal, 1)
		stop = make(chan struct{})
	)
	signal.Notify(kill, os.Interrupt)
	go func() {
		s := <-kill
		close(stop)
		klog.Infof("stopping... received signal %s", s)
	}()

	return stop
}

func clientsets() (kubernetes.Interface, crdv1alpha1.Interface, error) {
	k8sconfig, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set up K8s config from flags: %w", err)
	}

	k8s, err := kubernetes.NewForConfig(k8sconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize K8s clientsets: %w", err)
	}

	crd, err := crdv1alpha1.NewForConfig(k8sconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize CRD clientsets: %w", err)
	}

	return k8s, crd, err
}

func parseFlags() {
	klog.InitFlags(nil)

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.Parse()
}
