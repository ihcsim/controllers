package controller

import (
	"context"
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
	"k8s.io/apimachinery/pkg/labels"
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

	workerCount    int
	name           string
	ticker         *time.Ticker
	tickerDuration time.Duration

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
		name:                name,
		ticker:              time.NewTicker(time.Minute * 1),
		tickerDuration:      time.Minute * 1,
		workqueue:           workqueue,
		recorder:            recorder,
	}

	k8sInformers.Core().V1().Nodes().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{},
	)

	clusteropInformers.Clusterop().V1alpha1().KubeletUpgrades().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				now := metav1.Now()
				c.poll(obj, &now)
			},
			UpdateFunc: func(oldObj, newObj interface{}) {
				now := metav1.Now()
				c.poll(newObj, &now)
			},
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
	klog.Info("controller ready")

LOOP:
	for {
		select {
		case <-stop:
			klog.Info("shutting down workqueue")
			c.workqueue.ShutDown()
			break LOOP

		case <-c.ticker.C:
			klog.Info("polling KubeletConfig schedules")
			if err := c.pollSchedules(); err != nil {
				return err
			}
			klog.Infof("polling completed.. retrying in %s", c.tickerDuration)
		}
	}

	klog.Info("stopping controller")
	return nil
}

func (c *Controller) syncCache(stop <-chan struct{}) error {
	var (
		nodesSynced   = c.k8sInformers.Core().V1().Nodes().Informer().HasSynced
		upgradeSynced = c.clusteropInformers.Clusterop().V1alpha1().KubeletUpgrades().Informer().HasSynced
	)

	if ok := cache.WaitForCacheSync(stop, nodesSynced, upgradeSynced); !ok {
		return fmt.Errorf("cache sync'ed failed")
	}
	return nil
}

// pollSchedules checks the schedule of all the KubeletUpgrade objects to
// see if any upgrades are due. If they are, all the matching nodes will be added
// to the controller's workqueue, for further processing.
func (c *Controller) pollSchedules() error {
	upgrades, err := c.clusteropInformers.Clusterop().V1alpha1().KubeletUpgrades().Lister().List(labels.Everything())
	if err != nil {
		return err
	}

	if len(upgrades) == 0 {
		klog.Infof("no KubeletConfig objects found")
		return nil
	}

	var errs []error
	now := metav1.Now()
	for _, upgrade := range upgrades {
		// remains block until upgrade is completed to ensure stable and predictable
		// sequential upgrades
		if err := c.poll(upgrade, &now); err != nil {
			errs = append(errs, err)
		}
	}

	var final error
	for _, err := range errs {
		final = fmt.Errorf("%s\n%s", final, err)
	}

	return final
}

func (c *Controller) poll(obj interface{}, now *metav1.Time) error {
	upgrade, ok := obj.(*clusteropv1alpha1.KubeletUpgrade)
	if !ok {
		return fmt.Errorf("failed to poll KubeletUpgrade object. Unsupported type %T", obj)
	}

	c.updateNextScheduledTime(upgrade, now)
	if err := c.enqueueMatchingNodes(upgrade, now); err != nil {
		return err
	}

	// no more work to do since there are no matching nodes
	if c.workqueue.Len() == 0 {
		return nil
	}

	// block until all matching nodes are upgraded
	return c.startUpgrade(upgrade)
}

// updateNextScheduledTime updates the "next schedule time" property of the
// KubeletConfig obj status, and broadcast an event showing the result of the
// update.
func (c *Controller) updateNextScheduledTime(obj *clusteropv1alpha1.KubeletUpgrade, now *metav1.Time) {
	nextScheduledTime := obj.Status.NextScheduledTime
	if nextScheduledTime.IsZero() || nextScheduledTime.Before(now) {
		cloned := (*obj).UpdateNextScheduledTime(now)
		eventType := corev1.EventTypeNormal
		if _, err := c.clusteropClientsets.ClusteropV1alpha1().KubeletUpgrades().UpdateStatus(context.Background(), cloned, metav1.UpdateOptions{}); err != nil {
			eventType = corev1.EventTypeWarning
		}

		mostRecentCondition := cloned.Status.Conditions[len(cloned.Status.Conditions)-1]
		c.recorder.Event(cloned, eventType, mostRecentCondition.Reason, mostRecentCondition.Message)
	}
}

// enqueueMatchingNodes finds all the matching nodes of the KubeletUpgrade
// object and add them to the workqueue.
func (c *Controller) enqueueMatchingNodes(obj *clusteropv1alpha1.KubeletUpgrade, now *metav1.Time) error {
	nextScheduledTime := obj.Status.NextScheduledTime
	if now.Before(nextScheduledTime) || now.Equal(nextScheduledTime) {
		selector, err := metav1.LabelSelectorAsSelector(obj.Spec.Selector)
		if err != nil {
			return err
		}

		// exclude control plane nodes
		requirements, err := excludeRequirements()
		if err != nil {
			return err
		}
		selector = selector.Add(requirements...)

		nodes, err := c.k8sInformers.Core().V1().Nodes().Lister().List(selector)
		if err != nil {
			return err
		}

		if len(nodes) == 0 {
			klog.V(4).Infof("no matching nodes found for KubeletUpgrade %s", obj.GetName())
			return nil
		}

		var errs []error
		for _, node := range nodes {
			if err := c.enqueueNode(node); err != nil {
				errs = append(errs, err)
				continue
			}
			klog.V(4).Infof("added node %s to workqueue of %s", node.GetName(), obj.GetName())
		}

		var final error
		for _, err := range errs {
			final = fmt.Errorf("%s\n%s", final, err)
		}
		return final
	}

	return nil
}

func (c *Controller) enqueueNode(obj interface{}) error {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return err
	}

	if _, ok := obj.(*corev1.Node); !ok {
		return fmt.Errorf("failed to enqueue node %s. obj type must be *corev1.Node", key)
	}

	c.workqueue.Add(key)
	return nil
}

func (c *Controller) startUpgrade(upgrade *clusteropv1alpha1.KubeletUpgrade) error {
	var (
		wg         = sync.WaitGroup{}
		numWorkers = upgrade.Spec.MaxUnavailable
		numCurrent = 0
		errChan    = make(chan error, numWorkers)
		errs       error
	)

	defer close(errChan)
	go func() {
		for err := range errChan {
			errs = fmt.Errorf("%s. %s", errs, err)
		}
	}()

	for i := 0; numCurrent < numWorkers; i++ {
		klog.Infof("spawning worker %d", numCurrent)
		wg.Add(1)
		numCurrent++

		go func() {
			defer func() {
				klog.Infof("terminating worker %d", numCurrent)
				wg.Done()
			}()
			errChan <- c.runWorker()
		}()
	}
	wg.Wait()
	klog.Info("terminated all workers")

	return errs
}

func (c *Controller) runWorker() error {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return nil
	}
	defer c.workqueue.Done(obj)

	// obj is a string of the form namespace/name.
	// if it isn't, remove it from workqueue by calling Forget().
	key, ok := obj.(string)
	if !ok {
		c.workqueue.Forget(obj)
		return fmt.Errorf("unsupported workqueue key type %T. obj: %v", obj, obj)
	}

	klog.Infof("starting to upgrade %s", key)
	if result := c.upgradeKubelet(key); result.Err != nil {
		if errors.IsNotFound(result.Err) {
			klog.Infof("obj %s no longer exists; waiting for cache to sync", key)
			return nil
		}

		// re-queue node object for retries if error is transient
		if result.Retry {
			c.workqueue.AddRateLimited(key)
		}

		return fmt.Errorf("error upgrading %s: %w", key, result.Err)
	}
	klog.Infof("finish upgrading %s", key)
	return nil
}

func (c *Controller) upgradeKubelet(key string) *Result {
	result := &Result{}

	// convert the namespace/name key into its distinct namespace and name
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		result.Err = err
		return result
	}

	klog.Infof("getting KubeletUpgrade obj %s", name)
	obj, err := c.k8sInformers.Core().V1().Nodes().Lister().Get(name)
	if err != nil {
		result.Err = err
		return result
	}

	fmt.Printf("%+v\n", obj)

	return result
}
