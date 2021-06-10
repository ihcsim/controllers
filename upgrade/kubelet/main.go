package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/ihcsim/controllers/upgrade/kubelet/pkg/controller"
	clusteropclientsetv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/clientset/versioned"
	clusteropinformers "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/informers/externalversions"

	k8sinformers "k8s.io/client-go/informers"
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

	clientsetsK8s, clientsetsCRD, err := clientsets()
	if err != nil {
		klog.Exit(err)
	}
	factoryK8s, factoryCRD := informers(clientsetsK8s, clientsetsCRD)

	stop := handleSignal()
	c := controller.New(clientsetsK8s, clientsetsCRD, factoryK8s, factoryCRD)
	factoryK8s.Start(stop)
	factoryCRD.Start(stop)

	klog.Info("starting controller")
	if err := c.Sync(stop); err != nil {
		klog.Errorf("controller sync failed: ", err)
	}
	klog.Info("exiting")
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
		klog.Infof("shutting down due to signal %s", s)
	}()

	return stop
}

func clientsets() (kubernetes.Interface, clusteropclientsetv1alpha1.Interface, error) {
	k8sconfig, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to set up K8s config from flags: %w", err)
	}

	k8s, err := kubernetes.NewForConfig(k8sconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize K8s clientsets: %w", err)
	}

	crd, err := clusteropclientsetv1alpha1.NewForConfig(k8sconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize CRD clientsets: %w", err)
	}

	return k8s, crd, err
}

func informers(k8s kubernetes.Interface, crd clusteropclientsetv1alpha1.Interface) (k8sinformers.SharedInformerFactory, clusteropinformers.SharedInformerFactory) {
	resyncDuration := time.Minute * 10
	return k8sinformers.NewSharedInformerFactory(k8s, resyncDuration),
		clusteropinformers.NewSharedInformerFactory(crd, resyncDuration)
}

func parseFlags() {
	klog.InitFlags(nil)

	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.Parse()
}
