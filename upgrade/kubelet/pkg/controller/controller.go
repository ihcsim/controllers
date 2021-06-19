package controller

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	clusteropv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/apis/clusterop.isim.dev/v1alpha1"
	"github.com/ihcsim/controllers/upgrade/kubelet/pkg/controller/labels"
	clusteropclientsetsv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/clientset/versioned"
	"github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/clientset/versioned/scheme"
	clusteropinformers "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/informers/externalversions"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

var (
	tickDuration   = time.Minute * 1
	resyncDuration = time.Minute * 10
	requestTimeout = time.Second * 30
)

// Controller knows how to reconcile kubelet upgrades to match the in-cluster
// states with the desired states.
type Controller struct {
	k8sClientsets       kubernetes.Interface
	k8sInformers        k8sinformers.SharedInformerFactory
	clusteropClientsets clusteropclientsetsv1alpha1.Interface
	clusteropInformers  clusteropinformers.SharedInformerFactory

	workerCount int
	name        string
	ticker      *time.Ticker

	tickDuration   time.Duration
	requestTimeout time.Duration

	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	// workqueues maps KubeletUpgrade object names to their rate-limited queues.
	// The queues are used to store up nodes pending for kubelet upgrades.
	workqueues map[string]workqueue.RateLimitingInterface
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

	c := &Controller{
		k8sClientsets:       k8sClientsets,
		k8sInformers:        k8sInformers,
		clusteropClientsets: clusteropClientsets,
		clusteropInformers:  clusteropInformers,
		name:                name,
		ticker:              time.NewTicker(tickDuration),
		tickDuration:        tickDuration,
		requestTimeout:      requestTimeout,
		workqueues:          map[string]workqueue.RateLimitingInterface{},
		recorder:            recorder,
	}

	c.k8sInformers.Core().V1().Nodes().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{},
	)

	c.clusteropInformers.Clusterop().V1alpha1().KubeletUpgrades().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				now := time.Now()
				c.poll(obj, now)
			},
		},
	)

	return c
}

// Reconcile will continuously work towards reconciling the states of all the
// KubeletUpgrade objects. It starts the informers and sync their caches. It
// will block until stop is closed.
func (c *Controller) Reconcile(stop <-chan struct{}) error {
	defer func() {
		c.ticker.Stop()
		for _, workqueue := range c.workqueues {
			workqueue.ShutDown()
		}
		utilruntime.HandleCrash()
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
			break LOOP

		case <-c.ticker.C:
			klog.Info("polling KubeletConfig schedules")
			if err := c.pollSchedules(); err != nil {
				utilruntime.HandleError(fmt.Errorf("last schedules poll failed: %s", err))
				continue
			}
			klog.Infof("polling completed... retrying in %s", c.tickDuration)
		}
	}

	klog.Info("Controller stopped")
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
	requirements, err := labels.ExcludedKubeletUpgrade()
	if err != nil {
		return err
	}
	selector := k8slabels.NewSelector()
	selector = selector.Add(requirements...)
	upgrades, err := c.clusteropInformers.Clusterop().V1alpha1().KubeletUpgrades().Lister().List(selector)
	if err != nil {
		return err
	}

	if len(upgrades) == 0 {
		klog.Infof("no KubeletUpgrade objects found")
		return nil
	}

	var errs []error
	now := time.Now()
	for _, upgrade := range upgrades {
		// blocking calls to ensure stable, predictable sequential upgrade
		if err := c.poll(upgrade, now); err != nil {
			errs = append(errs, err)
		}
	}

	var final error
	for _, err := range errs {
		final = fmt.Errorf("%s\n%s", final, err)
	}

	return final
}

func (c *Controller) poll(obj interface{}, now time.Time) error {
	if !c.clusteropInformers.Clusterop().V1alpha1().KubeletUpgrades().Informer().HasSynced() ||
		!c.k8sInformers.Core().V1().Nodes().Informer().HasSynced() {
		return nil
	}

	upgrade, ok := obj.(*clusteropv1alpha1.KubeletUpgrade)
	if !ok {
		return fmt.Errorf("failed to poll %s: unrecognized type %T", obj, obj)
	}

	updatedClone, err := c.updateNextScheduledTime(upgrade, now, false)
	if err != nil {
		return err
	}

	proceed, reason := c.canUpgrade(updatedClone, now)
	if !proceed {
		klog.V(4).Infof("skipping upgrade (upgrade=%s,reason=%s)", updatedClone.GetName(), reason)
		return nil
	}

	if err := c.enqueueMatchingNodes(updatedClone); err != nil {
		return err
	}

	// queue is empty; nothing to do
	if c.queue(upgrade).Len() == 0 {
		return nil
	}

	errChan := make(chan error)
	go func() {
		defer close(errChan)
		errChan <- c.startUpgrade(updatedClone, now)
	}()

	return <-errChan
}

// updateNextScheduledTime updates the "next schedule time" property of the
// KubeletConfig obj status, and broadcast an event showing the result of the
// update.
func (c *Controller) updateNextScheduledTime(obj *clusteropv1alpha1.KubeletUpgrade, now time.Time, forceUpdate bool) (*clusteropv1alpha1.KubeletUpgrade, error) {
	next := obj.Status.NextScheduledTime.Time
	// account for the 5-minutes look-back window
	if next.IsZero() || forceUpdate || (next.Before(now) && now.Sub(next) >= time.Minute*5) {
		cloned := obj.UpdateNextScheduledTime(now)

		// broadcast event
		eventType := corev1.EventTypeNormal
		mostRecentCondition := cloned.Status.Conditions[len(cloned.Status.Conditions)-1]
		c.recorder.Event(cloned, eventType, mostRecentCondition.Reason, mostRecentCondition.Message)

		return c.clusteropClientsets.ClusteropV1alpha1().KubeletUpgrades().UpdateStatus(context.Background(), cloned, metav1.UpdateOptions{})
	}

	// no changes
	return obj, nil
}

// canUpgrade returns true if the KubeletUpgrade object satisfies all the
// upgrade preconditions:
// 1. the KubeletUpgrade object isn't disabled
// 2. the current system time is equal to or 5 minutes after the next scheduled
//    time
func (c *Controller) canUpgrade(obj *clusteropv1alpha1.KubeletUpgrade, now time.Time) (bool, string) {
	if obj.Labels != nil {
		if _, exists := obj.Labels[labels.KeyKubeletUpgradeSkip]; exists {
			return false, "has skip label"
		}
	}

	next := obj.Status.NextScheduledTime.Time
	if next.IsZero() {
		return false, "next scheduled time shouldn't be empty"
	}

	// look-back 5 minutes to see if any upgrades were missed
	if now.Equal(next) || (now.Sub(next) <= time.Minute*5 && now.Sub(next) >= 0) {
		return true, ""
	}

	return false, fmt.Sprintf("next scheduled time is %s", next.UTC())
}

// enqueueMatchingNodes finds all the matching nodes of the KubeletUpgrade
// object and add them to the workqueue.
func (c *Controller) enqueueMatchingNodes(obj *clusteropv1alpha1.KubeletUpgrade) error {
	selector, err := metav1.LabelSelectorAsSelector(&obj.Spec.Selector)
	if err != nil {
		return err
	}

	// exclude control plane nodes
	requirements, err := labels.ExcludedNodes()
	if err != nil {
		return err
	}
	selector = selector.Add(requirements...)
	nodes, err := c.k8sInformers.Core().V1().Nodes().Lister().List(selector)
	if err != nil {
		return err
	}

	var (
		errs      []error
		nodeNames []string
	)
	for _, node := range nodes {
		if err := c.enqueueNode(obj, node); err != nil {
			errs = append(errs, err)
			continue
		}
		nodeNames = append(nodeNames, node.GetName())
	}

	if len(nodeNames) > 0 {
		klog.Infof("matching nodes found: %v (upgrade=%s)", strings.Join(nodeNames, ", "), obj.GetName())
	}

	var final error
	for _, err := range errs {
		final = fmt.Errorf("%s. %s", final, err)
	}
	return final
}

func (c *Controller) enqueueNode(upgrade *clusteropv1alpha1.KubeletUpgrade, obj interface{}) error {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return err
	}

	if _, ok := obj.(*corev1.Node); !ok {
		return fmt.Errorf("failed to enqueue node %s due to wrong type %T", key, key)
	}

	c.queue(upgrade).Add(key)
	return nil
}

func (c *Controller) recordUpgradeStatus(
	obj *clusteropv1alpha1.KubeletUpgrade,
	status string,
	resultErr error,
	now time.Time) (*clusteropv1alpha1.KubeletUpgrade, error) {

	var (
		updatedClone *clusteropv1alpha1.KubeletUpgrade
		err          error
	)

	switch status {
	case "completed":
		updatedClone, err = obj.RecordUpgradeCompleted(err, now)
	case "started":
		updatedClone, err = obj.RecordUpgradeStarted(now)
	default:
		return nil, fmt.Errorf("unrecognized upgrade condition: %s", status)
	}
	if err != nil {
		return nil, err
	}

	// broadcast event
	eventType := corev1.EventTypeNormal
	if resultErr != nil {
		eventType = corev1.EventTypeWarning
	}
	mostRecentCondition := updatedClone.Status.Conditions[len(updatedClone.Status.Conditions)-1]
	c.recorder.Event(updatedClone, eventType, mostRecentCondition.Reason, mostRecentCondition.Message)

	rootCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	updateSpec := func() error {
		ctx, cancel := context.WithTimeout(rootCtx, c.requestTimeout)
		defer cancel()

		updatedClone, err = c.clusteropClientsets.ClusteropV1alpha1().KubeletUpgrades().Update(ctx, updatedClone, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
	}

	if err := updateSpec(); err != nil {
		return nil, err
	}

	updateStatus := func() error {
		ctx, cancel := context.WithTimeout(rootCtx, c.requestTimeout)
		defer cancel()

		updatedClone, err = c.clusteropClientsets.ClusteropV1alpha1().KubeletUpgrades().UpdateStatus(ctx, updatedClone, metav1.UpdateOptions{})
		if err != nil {
			return err
		}
		return nil
	}

	if err := updateStatus(); err != nil {
		return nil, err
	}

	return updatedClone, nil
}

// startUpgrade commences the kubelets upgrade process based on the spec of the
// KubeletUpgrade object. It remained blocked until all the kubelets are
// upgraded.
func (c *Controller) startUpgrade(obj *clusteropv1alpha1.KubeletUpgrade, now time.Time) error {
	var (
		numWorkers = obj.Spec.MaxUnavailable
		errChan    = make(chan error, numWorkers)
		errs       error
	)

	// essentially, we want number of workers to be
	// min(maxUnavailable, len(queue))
	queue := c.queue(obj)
	if queue.Len() < numWorkers {
		numWorkers = queue.Len()
	}

	// gather errors from all the workers
	go func() {
		for err := range errChan {
			if err != nil {
				errs = fmt.Errorf("%s. %s", errs, err)
			}
		}
	}()

	cloned := obj.DeepCopy()

	// record the 'start upgrade' condition
	updatedClone, err := c.recordUpgradeStatus(cloned, "started", nil, now)
	if err != nil {
		// ok to continue
		utilruntime.HandleError(fmt.Errorf("failed to update condition: %s (upgrade=%s)", err, cloned.GetName()))
	}

	// defer recording the 'end upgrade' condition
	defer func() {
		endTime := time.Now()
		updatedClone, err = c.recordUpgradeStatus(updatedClone, "completed", errs, endTime)
		if err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to update condition: %s (upgrade=%s)", err, updatedClone.GetName()))
		}

		if _, err := c.updateNextScheduledTime(updatedClone, now, true); err != nil {
			utilruntime.HandleError(fmt.Errorf("failed to update next scheduled time: %s (upgrade=%s)", err, updatedClone.GetName()))
		}
	}()

	wg := sync.WaitGroup{}
	for i := 0; i < queue.Len(); i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			errChan <- c.runWorker(workerID, updatedClone)
		}(i)
	}
	wg.Wait()
	close(errChan)

	return errs
}

func (c *Controller) queue(obj *clusteropv1alpha1.KubeletUpgrade) workqueue.RateLimitingInterface {
	if c.workqueues[obj.GetName()] == nil {
		c.workqueues[obj.GetName()] = workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			obj.GetName())
	}

	return c.workqueues[obj.GetName()]
}

// runWorker starts a worker to dequeue a node name from the controller's
// workqueue, and then upgrade the node's kubelet.
func (c *Controller) runWorker(workerID int, upgrade *clusteropv1alpha1.KubeletUpgrade) error {
	queue := c.queue(upgrade)
	obj, shutdown := queue.Get()
	if shutdown {
		return nil
	}
	defer queue.Done(obj)

	// obj must be a string of the form namespace/name, per enqueueNode().
	// if it isn't, remove it from workqueue by calling Forget().
	key, ok := obj.(string)
	if !ok {
		queue.Forget(obj)
		return fmt.Errorf("unsupported workqueue key type %T. node: %v (upgrade=%s,worker=#%d)", obj, obj, upgrade.GetName(), workerID)
	}

	klog.Infof("upgrading node %s (upgrade=%s,worker=#%d)", key, upgrade.GetName(), workerID)
	if result := c.upgradeKubelet(key, upgrade); result.Err != nil {
		if errors.IsNotFound(result.Err) {
			klog.Infof("node %s no longer exists", key)
			return nil
		}

		// re-queue node object for retries if error is transient
		if result.Retry {
			queue.AddRateLimited(key)
		}

		return fmt.Errorf("failed to upgrade node %s: %s (upgrade=%s,worker=#%d)", key, result.Err, upgrade.GetName(), workerID)
	}

	klog.Infof("finished upgrading node %s (upgrade=%s,worker=#%d)", key, upgrade.GetName(), workerID)
	return nil
}

func (c *Controller) upgradeKubelet(key string, upgrade *clusteropv1alpha1.KubeletUpgrade) *Result {
	result := &Result{}

	// convert the namespace/name key into its distinct namespace and name
	_, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		result.Err = err
		return result
	}

	node, err := c.k8sInformers.Core().V1().Nodes().Lister().Get(name)
	if err != nil {
		result.Err = err
		return result
	}

	updatedClone, err := c.cordonNode(true, *node, upgrade)
	if err != nil {
		result.Err = err
		return result
	}

	if _, err := c.cordonNode(false, *updatedClone, upgrade); err != nil {
		result.Err = err
		return result
	}

	return result
}

// cordonNode creates a clone of node and marks it as unschedulable, if the cordon
// flag is set to true. Othewise, it marks the node as schedulable.
func (c *Controller) cordonNode(cordon bool, node corev1.Node, upgrade *clusteropv1alpha1.KubeletUpgrade) (*corev1.Node, error) {
	action := "uncordoning"
	if cordon {
		action = "cordoning"
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.requestTimeout)
	defer cancel()

	klog.Infof("%s node (upgrade=%s,node=%s)", action, upgrade.GetName(), node.GetName())

	cloned := node.DeepCopy()
	cloned.Spec.Unschedulable = cordon
	return c.k8sClientsets.CoreV1().Nodes().Update(ctx, cloned, metav1.UpdateOptions{})
}
