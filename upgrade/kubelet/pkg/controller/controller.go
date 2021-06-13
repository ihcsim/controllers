package controller

import (
	"fmt"
	"sync"
	"time"

	clusteropv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/apis/clusterop.isim.dev/v1alpha1"
	clusteropclientsetsv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/clientset/versioned"
	"github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/clientset/versioned/scheme"
	clusteropinformers "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/informers/externalversions"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
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
	k8sClientsets       kubernetes.Interface
	k8sInformers        k8sinformers.SharedInformerFactory
	clusteropClientsets clusteropclientsetsv1alpha1.Interface
	clusteropInformers  clusteropinformers.SharedInformerFactory

	maxWorkersCount int
	name            string
	ticker          *time.Ticker

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// workqueue is a rate-limited queue that is used to process work
	// asynchronously.
	workqueue workqueue.RateLimitingInterface
}

// New returns a new instance of the controller.
func New(
	k8sClientsets kubernetes.Interface,
	k8sInformers k8sinformers.SharedInformerFactory,
	clusteropClientsets clusteropclientsetsv1alpha1.Interface,
	clusteropInformers clusteropinformers.SharedInformerFactory,
) *Controller {

	name := "kubelet-upgrade-controller"
	utilruntime.Must(clusteropv1alpha1.AddToScheme(scheme.Scheme))

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{
		Interface: k8sClientsets.CoreV1().Events(""),
	})
	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		corev1.EventSource{Component: name})

	workqueue := workqueue.NewNamedRateLimitingQueue(
		workqueue.DefaultControllerRateLimiter(),
		"kubeletUpgradeQueue")

	c := &Controller{
		k8sClientsets:       k8sClientsets,
		k8sInformers:        k8sInformers,
		clusteropClientsets: clusteropClientsets,
		clusteropInformers:  clusteropInformers,
		maxWorkersCount:     1,
		name:                name,
		ticker:              time.NewTicker(time.Second),
		workqueue:           workqueue,
		recorder:            recorder,
	}

	k8sInformers.Core().V1().Nodes().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{},
	)

	clusteropInformers.Clusterop().V1alpha1().KubeletUpgrades().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: c.enqueue,
			UpdateFunc: func(oldobj, newObj interface{}) {
				c.enqueue(newObj)
			},
			DeleteFunc: c.purge,
		},
	)

	return c
}

// Run will set up the event handlers for our custom resources, as well as
// syncing informer caches and starting workers. It will block until stopCh is
// closed, at which point it will shutdown the workqueue and wait for
func (c *Controller) Run(stop <-chan struct{}) error {
	defer func() {
		utilruntime.HandleCrash()
		c.workqueue.ShutDown()
		c.ticker.Stop()
	}()

	c.k8sInformers.Start(stop)
	c.clusteropInformers.Start(stop)
	if err := c.syncCache(stop); err != nil {
		return err
	}

	var (
		currentWorkersCount = 0
		waitGroup           = sync.WaitGroup{}
	)

LOOP:
	for {
		select {
		case <-stop:
			klog.Info("shutting down workqueue")
			c.workqueue.ShutDown()
			break LOOP

		case <-c.ticker.C:
			if currentWorkersCount <= c.maxWorkersCount {
				waitGroup.Add(1)
				currentWorkersCount++

				go func() {
					defer func() {
						waitGroup.Done()
						currentWorkersCount--
					}()
					c.dequeue(stop)
				}()
			}
		}
	}
	waitGroup.Wait()
	klog.Info("stopping controller")

	return nil
}

func (c *Controller) syncCache(stop <-chan struct{}) error {
	var (
		nodesSynced   = c.k8sInformers.Core().V1().Nodes().Informer().HasSynced
		upgradeSynced = c.clusteropInformers.Clusterop().V1alpha1().KubeletUpgrades().Informer().HasSynced
	)

	klog.Info("waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stop, nodesSynced, upgradeSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}
	klog.Info("cache sync completed")
	return nil
}

func (c *Controller) enqueue(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	if _, ok := obj.(*clusteropv1alpha1.KubeletUpgrade); !ok {
		klog.Errorf("failed to enqueue obj %s. reason: wrong type", key)
		return
	}

	c.workqueue.Add(key)
	klog.Infof("added KubeletUpgrade obj %s", key)
}

func (c *Controller) dequeue(stop <-chan struct{}) {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return
	}

	// obj is a string of the form namespace/name
	key, ok := obj.(string)
	if !ok {
		utilruntime.HandleError(fmt.Errorf("unsupported workqueue key type %T. obj: %v", obj, obj))
		return
	}

	defer c.workqueue.Done(obj)
	if err := c.sync(key); err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("obj %s no longer exists", key)
			return
		}

		utilruntime.HandleError(fmt.Errorf("error syncing: %w", err))
		return
	}
	klog.Infof("upgrade %s completed successfully", key)

	// re-queue upgrade object for future processing
	c.workqueue.AddRateLimited(key)
}

func (c *Controller) purge(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(err)
		return
	}

	c.workqueue.Done(key)
	c.workqueue.Forget(key)
	klog.Infof("deleted KubeletUpgrade obj %s", key)
}

func (c *Controller) sync(key string) error {
	// convert the namespace/name key into its distinct namespace and name
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	klog.Infof("getting KubeletUpgrade obj %s", name)
	obj, err := c.clusteropInformers.Clusterop().V1alpha1().KubeletUpgrades().Lister().Get(name)
	if err != nil {
		return err
	}

	nextScheduledTime := obj.Status.NextScheduledTime
	if nextScheduledTime.IsZero() {
		return nil
	}

	now := metav1.Now()
	if !obj.Status.NextScheduledTime.Before(&now) {
		return nil
	}

	if err := upgradeKubelet(); err != nil {
		return err
	}

	return nil
}

func next() metav1.Time {
	return metav1.NewTime(time.Now().Add(time.Minute))
}

func upgradeKubelet() error {
	return nil
}
