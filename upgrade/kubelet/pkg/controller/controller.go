package controller

import (
	"fmt"
	"time"

	clusteropv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/apis/clusterop.isim.dev/v1alpha1"
	clusteropclientsetsv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/clientset/versioned"
	"github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/clientset/versioned/scheme"
	clusteropinformers "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/informers/externalversions"

	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

// Controller knows how to reconcile kubelet upgrades to match the in-cluster
// states with the desired states.
type Controller struct {
	clientsetsK8s      kubernetes.Interface
	clientsetsUpgrade  clusteropclientsetsv1alpha1.Interface
	k8sinformers       k8sinformers.SharedInformerFactory
	clusteropinformers clusteropinformers.SharedInformerFactory

	name string

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// workqueue is a rate-limited queue that is used to process work
	// asynchronously.
	workqueue workqueue.RateLimitingInterface
}

// New returns a new instance of the controller.
func New(
	clientsetsK8s kubernetes.Interface,
	clientsetsCRD clusteropclientsetsv1alpha1.Interface,
	k8sinformers k8sinformers.SharedInformerFactory,
	clusteropinformers clusteropinformers.SharedInformerFactory,
) *Controller {

	name := "kubelet-upgrade-controller"
	utilruntime.Must(clusteropv1alpha1.AddToScheme(scheme.Scheme))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: clientsetsK8s.CoreV1().Events(""),
	})
	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		corev1.EventSource{Component: name})

	workqueue := workqueue.NewNamedRateLimitingQueue(
		workqueue.DefaultControllerRateLimiter(),
		"kubeletUpgradeQueue")

	c := &Controller{
		clientsetsK8s:      clientsetsK8s,
		clientsetsUpgrade:  clientsetsCRD,
		k8sinformers:       k8sinformers,
		clusteropinformers: clusteropinformers,
		name:               name,
		workqueue:          workqueue,
		recorder:           recorder,
	}

	clusteropinformers.Clusterop().V1alpha1().KubeletUpgradeConfigs().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: c.enqueue,
			UpdateFunc: func(oldobj, newobj interface{}) {
				c.enqueue(newobj)
			},
		},
	)

	return c
}

// Sync will set up the event handlers for our custom resources, as well as
// syncing informer caches and starting workers. It will block until stopCh is
// closed, at which point it will shutdown the workqueue and wait for
func (c *Controller) Sync(stop <-chan struct{}) error {
	defer func() {
		utilruntime.HandleCrash()
		c.workqueue.ShutDown()
	}()

	var (
		nodesSynced = c.k8sinformers.Core().V1().Nodes().Informer().HasSynced
		podsSynced  = c.k8sinformers.Core().V1().Pods().Informer().HasSynced
		crdSynced   = c.clusteropinformers.Clusterop().V1alpha1().KubeletUpgradeConfigs().Informer().HasSynced
	)
	klog.Info("waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stop, nodesSynced, podsSynced, crdSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("starting workers")
	for i := 0; i < 10; i++ {
		go wait.Until(c.run, time.Second, stop)
	}

	<-stop
	klog.Info("shutting down workers")
	return nil
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	if _, ok := obj.(*clusteropv1alpha1.KubeletUpgradeConfig); !ok {
		klog.Error("failed to enqueue obj %s. reason: wrong type", key)
		return
	}

	c.workqueue.Add(key)
}

func (c *Controller) run() {}
