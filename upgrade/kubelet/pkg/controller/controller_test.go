package controller

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	"github.com/ihcsim/controllers/crd/pkg/envtest"
	clusteropv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/apis/clusterop.isim.dev/v1alpha1"
	"github.com/ihcsim/controllers/upgrade/kubelet/pkg/controller/labels"
	clusteropv1alpha1testing "github.com/ihcsim/controllers/upgrade/kubelet/pkg/controller/testing"
	clusteropclientsetv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/clientset/versioned"
	"github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/clientset/versioned/scheme"
	clusteropinformers "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/informers/externalversions"
)

var (
	binDir       = "../../../bin"
	crdDir       = "../../../kubelet/yaml/crd"
	testEnvDebug = false
	stop         = make(chan struct{})

	testController *Controller
	fakeRecorder   *record.FakeRecorder

	emptyConditions = []clusteropv1alpha1.UpgradeCondition{}
)

func TestMain(m *testing.M) {
	// start the test API server and etcd
	k8sconfig, err := envtest.Start(binDir, crdDir, testEnvDebug)
	if err != nil {
		log.Println(err)
		os.Exit(1)
	}
	defer func() {
		envtest.Stop()
		close(fakeRecorder.Events)
		close(stop)
	}()

	if err := initTestController(k8sconfig); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	if err := syncTestControllerCache(stop); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func initTestController(k8sconfig *rest.Config) error {
	utilruntime.Must(clusteropv1alpha1.AddToScheme(scheme.Scheme))

	k8sClientsets, err := kubernetes.NewForConfig(k8sconfig)
	if err != nil {
		return err
	}
	k8sInformers := k8sinformers.NewSharedInformerFactory(k8sClientsets, resyncDuration)

	clusteropClientsets, err := clusteropclientsetv1alpha1.NewForConfig(k8sconfig)
	if err != nil {
		return err
	}
	clusteropInformers := clusteropinformers.NewSharedInformerFactory(clusteropClientsets, resyncDuration)

	name := "test-controller"
	testController = &Controller{
		k8sClientsets:       k8sClientsets,
		k8sInformers:        k8sInformers,
		clusteropClientsets: clusteropClientsets,
		clusteropInformers:  clusteropInformers,
		name:                name,
		ticker:              time.NewTicker(tickDuration),
		tickDuration:        tickDuration,
		requestTimeout:      requestTimeout,
		workqueues:          map[string]workqueue.RateLimitingInterface{},
	}

	testController.recorder = record.NewFakeRecorder(10)
	fakeRecorder = testController.recorder.(*record.FakeRecorder)

	testController.k8sInformers.Core().V1().Nodes().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{},
	)

	testController.clusteropInformers.Clusterop().V1alpha1().KubeletUpgrades().Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{},
	)

	return nil
}

func syncTestControllerCache(stop <-chan struct{}) error {
	testController.k8sInformers.Start(stop)
	testController.clusteropInformers.Start(stop)
	return testController.syncCache(stop)
}

func TestUpdateNextScheduledTime(t *testing.T) {
	type testCase struct {
		now               time.Time
		nextScheduledTime time.Time
		expected          time.Time
		updateStatus      bool // some attributes require calls to UpdateStatus() before executing the test case
		forceUpdate       bool
		expectedEvent     *corev1.Event
	}

	// call this function to test and assert a test case
	assertTestCase := func(tc testCase, objName string) {
		created, err := applyTestKubeletUpgrade(objName, "@every 5m", nil)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		if tc.updateStatus {
			created, err = applyTestKubeletUpgradeStatus(created, tc.nextScheduledTime, emptyConditions)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		}

		updated, err := testController.updateNextScheduledTime(created, tc.now, tc.forceUpdate)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		actual := updated.Status.NextScheduledTime.Time
		if !tc.expected.Equal(actual) {
			t.Errorf("mismatch next scheduled time. Expected: %s.   Actual: %s", tc.expected, actual)
		}

		if e := tc.expectedEvent; e != nil {
			// see https://github.com/kubernetes/client-go/blob/58854425ecd20b43fd398e2b5d75d4fb5f323af3/tools/record/fake.go#L46
			expected := fmt.Sprintf("%s %s %s", e.Type, e.Reason, e.Message)
			actual := <-fakeRecorder.Events
			if expected != actual {
				t.Errorf("mismatch events.\n  Expected: %s\n  Actual: %s", expected, actual)
			}
		}
	}

	t.Run("scheduled time expired", func(t *testing.T) {
		var testCases = []testCase{
			{
				now:      time.Date(2021, 6, 17, 13, 46, 0, 0, time.UTC),
				expected: time.Date(2021, 6, 17, 13, 51, 0, 0, time.UTC),
				expectedEvent: &corev1.Event{
					Type:   corev1.EventTypeNormal,
					Reason: clusteropv1alpha1.ConditionReasonNextScheduledTimeStale,
					Message: fmt.Sprintf(
						clusteropv1alpha1.ConditionMessageUpdateNextScheduleTime,
						time.Date(2021, 6, 17, 13, 51, 0, 0, time.UTC)),
				},
			},
			{
				now:               time.Date(2021, 6, 17, 13, 46, 0, 0, time.UTC),
				nextScheduledTime: time.Date(2021, 6, 17, 13, 36, 0, 0, time.UTC),
				expected:          time.Date(2021, 6, 17, 13, 51, 0, 0, time.UTC),
				updateStatus:      true,
				expectedEvent: &corev1.Event{
					Type:   corev1.EventTypeNormal,
					Reason: clusteropv1alpha1.ConditionReasonNextScheduledTimeStale,
					Message: fmt.Sprintf(
						clusteropv1alpha1.ConditionMessageUpdateNextScheduleTime,
						time.Date(2021, 6, 17, 13, 51, 0, 0, time.UTC)),
				},
			},
			{
				now:               time.Date(2021, 6, 17, 23, 55, 0, 0, time.UTC),
				nextScheduledTime: time.Date(2021, 6, 17, 23, 45, 0, 0, time.UTC),
				expected:          time.Date(2021, 6, 18, 0, 0, 0, 0, time.UTC),
				updateStatus:      true,
				expectedEvent: &corev1.Event{
					Type:   corev1.EventTypeNormal,
					Reason: clusteropv1alpha1.ConditionReasonNextScheduledTimeStale,
					Message: fmt.Sprintf(
						clusteropv1alpha1.ConditionMessageUpdateNextScheduleTime,
						time.Date(2021, 6, 18, 0, 0, 0, 0, time.UTC)),
				},
			},
		}

		for i, tc := range testCases {
			t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
				objName := fmt.Sprintf("kubelet-upgrade-expired-%d", i)
				assertTestCase(tc, objName)
			})
		}
	})

	t.Run("scheduled time expired but within look-back window", func(t *testing.T) {
		var testCases = []testCase{
			{
				now:               time.Date(2021, 6, 17, 13, 46, 0, 0, time.UTC),
				nextScheduledTime: time.Date(2021, 6, 17, 13, 44, 0, 0, time.UTC),
				expected:          time.Date(2021, 6, 17, 13, 44, 0, 0, time.UTC),
				updateStatus:      true,
			},
			{
				now:               time.Date(2021, 6, 17, 23, 55, 0, 0, time.UTC),
				nextScheduledTime: time.Date(2021, 6, 17, 23, 51, 0, 0, time.UTC),
				expected:          time.Date(2021, 6, 17, 23, 51, 0, 0, time.UTC),
				updateStatus:      true,
			},
		}

		for i, tc := range testCases {
			t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
				objName := fmt.Sprintf("kubelet-upgrade-expired-with-look-back-%d", i)
				assertTestCase(tc, objName)
			})
		}
	})

	t.Run("scheduled time not expired", func(t *testing.T) {
		var testCases = []testCase{
			{
				now:               time.Date(2021, 6, 17, 13, 46, 0, 0, time.UTC),
				nextScheduledTime: time.Date(2021, 6, 17, 13, 47, 0, 0, time.UTC),
				expected:          time.Date(2021, 6, 17, 13, 47, 0, 0, time.UTC),
				updateStatus:      true,
			},
			{
				now:               time.Date(2021, 6, 17, 13, 46, 0, 0, time.UTC),
				nextScheduledTime: time.Date(2021, 6, 17, 13, 56, 0, 0, time.UTC),
				expected:          time.Date(2021, 6, 17, 13, 56, 0, 0, time.UTC),
				updateStatus:      true,
			},
			{
				now:               time.Date(2021, 6, 17, 23, 46, 0, 0, time.UTC),
				nextScheduledTime: time.Date(2021, 6, 18, 00, 06, 0, 0, time.UTC),
				expected:          time.Date(2021, 6, 18, 00, 06, 0, 0, time.UTC),
				updateStatus:      true,
			},
		}

		for i, tc := range testCases {
			t.Run(fmt.Sprintf("test case %d", i), func(t *testing.T) {
				objName := fmt.Sprintf("kubelet-upgrade-not-expired-%d", i)
				assertTestCase(tc, objName)
			})
		}
	})
}

func TestCanUpgrade(t *testing.T) {
	t.Run("based on 'skip' label", func(t *testing.T) {
		type testCase struct {
			description    string
			labels         map[string]string
			expected       bool
			expectedReason string
		}

		// call this function to test and assert a test case
		assertTestCase := func(tc testCase, objName string) {
			created, err := applyTestKubeletUpgrade(objName, "@every 5m", tc.labels)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			actual, actualReason := testController.canUpgrade(created, time.Time{})
			if tc.expected != actual {
				t.Errorf("mismatch result. Expected: %t. Actual: %t", tc.expected, actual)
			}

			if tc.expectedReason != actualReason {
				t.Errorf("mismatch reasons. Expected: %s. Actual: %s", tc.expectedReason, actualReason)
			}
		}

		testCases := []testCase{
			{
				description: "has skip labels",
				labels: map[string]string{
					labels.KeyKubeletUpgradeSkip: "",
				},
				expected:       false,
				expectedReason: "has skip label",
			},
		}

		for i, tc := range testCases {
			t.Run(tc.description, func(t *testing.T) {
				objName := fmt.Sprintf("kubelet-upgrade-label-%d", i)
				assertTestCase(tc, objName)
			})
		}
	})

	t.Run("based on next scheduled time", func(t *testing.T) {
		type testCase struct {
			description       string
			nextScheduledTime time.Time
			now               time.Time
			updateStatus      bool // some attributes required object status to be updated to work
			expected          bool
			expectedReason    string
		}

		// call this function to test and assert a test case
		assertTestCase := func(tc testCase, objName string) {
			created, err := applyTestKubeletUpgrade(objName, "@every 5m", nil)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if tc.updateStatus {
				created, err = applyTestKubeletUpgradeStatus(created, tc.nextScheduledTime, emptyConditions)
				if err != nil {
					t.Fatalf("unexpected error: %s", err)
				}
			}

			actual, actualReason := testController.canUpgrade(created, tc.now)
			if tc.expected != actual {
				t.Errorf("mismatch result. Expected: %t. Actual: %t", tc.expected, actual)
			}

			if tc.expectedReason != actualReason {
				t.Errorf("mismatch reasons. Expected: %s. Actual: %s", tc.expectedReason, actualReason)
			}
		}

		t.Run("can't upgrade", func(t *testing.T) {
			testCases := []testCase{
				{
					description:    "next scheduled time is not set",
					now:            time.Date(2021, 6, 1, 14, 0, 0, 0, time.UTC),
					expected:       false,
					expectedReason: "next scheduled time shouldn't be empty",
				},
				{
					description:       "next scheduled time 1 minute in the future",
					nextScheduledTime: time.Date(2021, 11, 10, 23, 1, 0, 0, time.UTC),
					now:               time.Date(2021, 11, 10, 23, 0, 0, 0, time.UTC),
					updateStatus:      true,
					expected:          false,
					expectedReason: fmt.Sprintf("next scheduled time is %s",
						time.Date(2021, 11, 10, 23, 1, 0, 0, time.UTC)),
				},
				{
					description:       "next scheduled time is in the future",
					nextScheduledTime: time.Date(2021, 11, 10, 23, 0, 0, 0, time.UTC),
					now:               time.Date(2021, 6, 1, 14, 0, 0, 0, time.UTC),
					updateStatus:      true,
					expected:          false,
					expectedReason: fmt.Sprintf("next scheduled time is %s",
						time.Date(2021, 11, 10, 23, 0, 0, 0, time.UTC)),
				},
				{
					description:       "next scheduled time is 1 minute past the look-back window",
					nextScheduledTime: time.Date(2020, 6, 1, 13, 55, 0, 0, time.UTC),
					now:               time.Date(2021, 6, 1, 14, 0, 0, 0, time.UTC),
					updateStatus:      true,
					expected:          false,
					expectedReason: fmt.Sprintf("next scheduled time is %s",
						time.Date(2020, 6, 1, 13, 55, 0, 0, time.UTC)),
				},
				{
					description:       "next scheduled time is in the past",
					nextScheduledTime: time.Date(2020, 11, 10, 23, 0, 0, 0, time.UTC),
					now:               time.Date(2021, 6, 1, 14, 0, 0, 0, time.UTC),
					updateStatus:      true,
					expected:          false,
					expectedReason: fmt.Sprintf("next scheduled time is %s",
						time.Date(2020, 11, 10, 23, 0, 0, 0, time.UTC)),
				},
			}

			for i, tc := range testCases {
				t.Run(tc.description, func(t *testing.T) {
					objName := fmt.Sprintf("kubelet-cannot-upgrade-%d", i)
					assertTestCase(tc, objName)
				})
			}
		})

		t.Run("can upgrade", func(t *testing.T) {
			testCases := []testCase{
				{
					description:       "next scheduled time equals 'now'",
					nextScheduledTime: time.Date(2021, 11, 10, 23, 0, 0, 0, time.UTC),
					now:               time.Date(2021, 11, 10, 23, 0, 0, 0, time.UTC),
					updateStatus:      true,
					expected:          true,
				},
				{
					description:       "next scheduled time is 1 min behind 'now'",
					nextScheduledTime: time.Date(2021, 11, 10, 18, 59, 0, 0, time.UTC),
					now:               time.Date(2021, 11, 10, 19, 00, 0, 0, time.UTC),
					updateStatus:      true,
					expected:          true,
				},
				{
					description:       "next scheduled time is 2 mins behind 'now'",
					nextScheduledTime: time.Date(2021, 11, 10, 18, 58, 0, 0, time.UTC),
					now:               time.Date(2021, 11, 10, 19, 00, 0, 0, time.UTC),
					updateStatus:      true,
					expected:          true,
				},
				{
					description:       "next scheduled time is 3 mins behind 'now'",
					nextScheduledTime: time.Date(2021, 11, 10, 18, 57, 0, 0, time.UTC),
					now:               time.Date(2021, 11, 10, 19, 00, 0, 0, time.UTC),
					updateStatus:      true,
					expected:          true,
				},
				{
					description:       "next scheduled time is 4 mins behind 'now'",
					nextScheduledTime: time.Date(2021, 11, 10, 18, 56, 0, 0, time.UTC),
					now:               time.Date(2021, 11, 10, 19, 00, 0, 0, time.UTC),
					updateStatus:      true,
					expected:          true,
				},
			}

			for i, tc := range testCases {
				t.Run(tc.description, func(t *testing.T) {
					objName := fmt.Sprintf("kubelet-can-upgrade-%d", i)
					assertTestCase(tc, objName)
				})
			}
		})
	})
}

func TestEnqueueMatchingNodes(t *testing.T) {
	var testCases = []struct {
		name          string
		selector      metav1.LabelSelector
		expectedNodes []string
	}{
		{
			name: "kubelet-upgrade-node-pool-0",
			selector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "node-pool",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"0"},
					},
				},
			},
			expectedNodes: []string{"worker-0", "worker-1"},
		},
		{
			name: "kubelet-upgrade-node-pool-1",
			selector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "node-pool",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"1"},
					},
				},
			},
			expectedNodes: []string{"worker-2"},
		},
		{
			name: "kubelet-upgrade-node-pool-2",
			selector: metav1.LabelSelector{
				MatchExpressions: []metav1.LabelSelectorRequirement{
					{
						Key:      "node-pool",
						Operator: metav1.LabelSelectorOpIn,
						Values:   []string{"2"},
					},
				},
			},
			expectedNodes: []string{"worker-3"},
		},
	}

	if err := applyTestNodes(); err != nil {
		t.Fatal("unexpected error: ", err)
	}

	for _, tc := range testCases {
		obj := &clusteropv1alpha1.KubeletUpgrade{
			ObjectMeta: metav1.ObjectMeta{
				Name: tc.name,
			},
			Spec: clusteropv1alpha1.KubeletUpgradeSpec{
				Selector: tc.selector,
			},
		}

		if err := testController.enqueueMatchingNodes(obj); err != nil {
			t.Fatal("unexpected error: ", err)
		}
	}

	numNodePools := 3
	if len(testController.workqueues) != numNodePools {
		t.Fatalf("expected number of workqueues to be %d, but got %d", numNodePools, len(testController.workqueues))
	}

	// dequeue nodes from queue and store them in 'actuals'
	actuals := map[string][]string{}
	for i := 0; i < numNodePools; i++ {
		kubeletUpgrade := fmt.Sprintf("kubelet-upgrade-node-pool-%d", i)
		if _, exists := actuals[kubeletUpgrade]; !exists {
			actuals[kubeletUpgrade] = []string{}
		}

		workqueue, exists := testController.workqueues[kubeletUpgrade]
		if !exists {
			t.Errorf("expected workqueue for obj %s to exist", kubeletUpgrade)
		}

		count := workqueue.Len()
		for j := 0; j < count; j++ {
			node, _ := workqueue.Get()
			actuals[kubeletUpgrade] = append(actuals[kubeletUpgrade], node.(string))
			workqueue.Done(node)
		}
	}

	// test assertion
	for _, tc := range testCases {
		actualNodes, exists := actuals[tc.name]
		if !exists {
			t.Fatalf("expected nodes to exist (upgrade=%s)", tc.name)
		}

		for _, expected := range tc.expectedNodes {
			var found bool
			for _, actual := range actualNodes {
				if expected == actual {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("expected node %s to exist in queue (upgrade=%s)", expected, tc.name)
			}
		}
	}
}

func TestRecordUpgradeStatus(t *testing.T) {
	var (
		now               = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
		nextScheduledTime = time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	)

	var testCases = []struct {
		description       string
		expectedCondition clusteropv1alpha1.UpgradeCondition
		expectedEvent     *corev1.Event
		resultErr         error
		status            string
	}{
		{
			description: "started",
			status:      "started",
			expectedCondition: clusteropv1alpha1.UpgradeCondition{
				Type:    clusteropv1alpha1.ConditionTypeUpgradeStarted,
				Message: clusteropv1alpha1.ConditionMessageUpgradeStarted,
				Status:  clusteropv1alpha1.ConditionStatusStarted,
			},
			expectedEvent: &corev1.Event{
				Message: clusteropv1alpha1.ConditionMessageUpgradeStarted,
				Reason:  clusteropv1alpha1.ConditionStatusStarted,
				Type:    corev1.EventTypeNormal,
			},
		},
		{
			description: "completed",
			status:      "completed",
			expectedCondition: clusteropv1alpha1.UpgradeCondition{
				Type:    clusteropv1alpha1.ConditionTypeUpgradeCompleted,
				Message: clusteropv1alpha1.ConditionMessageUpgradeCompleted,
				Status:  clusteropv1alpha1.ConditionStatusCompleted,
			},
			expectedEvent: &corev1.Event{
				Message: clusteropv1alpha1.ConditionMessageUpgradeCompleted,
				Reason:  clusteropv1alpha1.ConditionStatusCompleted,
				Type:    corev1.EventTypeNormal,
			},
		},
		{
			description: "completed with errors",
			status:      "completed",
			resultErr:   errors.New("failed test upgrade"),
			expectedCondition: clusteropv1alpha1.UpgradeCondition{
				Type:    clusteropv1alpha1.ConditionTypeUpgradeCompleted,
				Message: "failed test upgrade",
				Status:  clusteropv1alpha1.ConditionStatusFailed,
			},
			expectedEvent: &corev1.Event{
				Reason:  clusteropv1alpha1.ConditionStatusFailed,
				Message: "failed test upgrade",
				Type:    corev1.EventTypeWarning,
			},
		},
	}

	for i, tc := range testCases {
		objName := fmt.Sprintf("kubelet-upgrade-record-status-%d", i)
		created, err := applyTestKubeletUpgrade(objName, "@every 5m", nil)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		created, err = applyTestKubeletUpgradeStatus(created, nextScheduledTime, emptyConditions)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
		}

		updated, err := testController.recordUpgradeStatus(created, tc.status, tc.resultErr, now)
		if err != nil {
			t.Fatal("unexpected error: ", err)
		}

		if actual := len(updated.Status.Conditions); actual != 1 {
			t.Fatalf("expected upgrade object %q to have 1 status condition, but got %d", objName, actual)
		}

		if e := tc.expectedEvent; e != nil {
			// see https://github.com/kubernetes/client-go/blob/58854425ecd20b43fd398e2b5d75d4fb5f323af3/tools/record/fake.go#L46
			expected := fmt.Sprintf("%s %s %s", e.Type, e.Reason, e.Message)
			actual := <-fakeRecorder.Events
			if expected != actual {
				t.Errorf("mismatch events.\n  Expected: %s\n  Actual: %s", expected, actual)
			}
		}
	}
}

func applyTestKubeletUpgrade(name, schedule string, labels map[string]string) (*clusteropv1alpha1.KubeletUpgrade, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testController.requestTimeout)
	defer cancel()

	obj := clusteropv1alpha1testing.NewTestKubeletUpgrade(name)
	clusteropv1alpha1testing.WithSchedule(obj, schedule)
	clusteropv1alpha1testing.WithLabels(obj, labels)
	return testController.clusteropClientsets.ClusteropV1alpha1().KubeletUpgrades().Create(ctx, obj, metav1.CreateOptions{})
}

func applyTestKubeletUpgradeStatus(obj *clusteropv1alpha1.KubeletUpgrade, nextScheduledTime time.Time, conditions []clusteropv1alpha1.UpgradeCondition) (*clusteropv1alpha1.KubeletUpgrade, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testController.requestTimeout)
	defer cancel()

	clusteropv1alpha1testing.WithStatusConditions(obj, conditions)
	clusteropv1alpha1testing.WithNextScheduledTime(obj, nextScheduledTime)

	return testController.clusteropClientsets.ClusteropV1alpha1().KubeletUpgrades().UpdateStatus(ctx, obj, metav1.UpdateOptions{})
}

func applyTestNodes() error {
	var nodes = []struct {
		name   string
		labels map[string]string
	}{
		{
			name:   "control-plane-0",
			labels: map[string]string{"node-role.kubernetes.io/control-plane": ""},
		},
		{
			name:   "control-plane-1",
			labels: map[string]string{"node-role.kubernetes.io/control-plane": ""},
		},
		{
			name:   "control-plane-2",
			labels: map[string]string{"node-role.kubernetes.io/master": ""},
		},
		{
			name:   "worker-0",
			labels: map[string]string{"node-pool": "0"},
		},
		{
			name:   "worker-1",
			labels: map[string]string{"node-pool": "0"},
		},
		{
			name:   "worker-2",
			labels: map[string]string{"node-pool": "1"},
		},
		{
			name:   "worker-3",
			labels: map[string]string{"node-pool": "2"},
		},
	}

	for _, node := range nodes {
		if _, err := applyTestNode(node.name, node.labels); err != nil {
			return err
		}
	}

	return nil
}

func applyTestNode(name string, labels map[string]string) (*corev1.Node, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testController.requestTimeout)
	defer cancel()

	// create the object
	obj := clusteropv1alpha1testing.NewNode(name)
	clusteropv1alpha1testing.NodeWithLabels(obj, labels)
	return testController.k8sClientsets.CoreV1().Nodes().Create(ctx, obj, metav1.CreateOptions{})
}
