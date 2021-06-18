package controller

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"

	"github.com/ihcsim/controllers/crd/pkg/envtest"
	clusteropv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/apis/clusterop.isim.dev/v1alpha1"
	"github.com/ihcsim/controllers/upgrade/kubelet/pkg/controller/labels"
	clusteropv1alpha1testing "github.com/ihcsim/controllers/upgrade/kubelet/pkg/controller/testing"
	clusteropclientsetv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/clientset/versioned"
	clusteropinformers "github.com/ihcsim/controllers/upgrade/kubelet/pkg/generated/informers/externalversions"
)

var (
	binDir         = "../../../bin"
	crdDir         = "../../../kubelet/yaml/crd"
	testEnvDebug   = false
	testController *Controller
	fakeRecorder   *record.FakeRecorder
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
	}()

	if err := initTestController(k8sconfig); err != nil {
		log.Println(err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func initTestController(k8sconfig *rest.Config) error {
	k8sClientsets, err := kubernetes.NewForConfig(k8sconfig)
	if err != nil {
		return err
	}

	clusteropClientsets, err := clusteropclientsetv1alpha1.NewForConfig(k8sconfig)
	if err != nil {
		return err
	}

	k8sInformers := k8sinformers.NewSharedInformerFactory(k8sClientsets, time.Minute*10)
	clusteropInformers := clusteropinformers.NewSharedInformerFactory(clusteropClientsets, time.Minute*10)

	testController = New(k8sClientsets, k8sInformers, clusteropClientsets, clusteropInformers)
	testController.recorder = record.NewFakeRecorder(10)
	fakeRecorder = testController.recorder.(*record.FakeRecorder)
	return nil
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
		created, err := applyTestObj(objName, "@every 5m", tc.updateStatus, tc.nextScheduledTime, nil)
		if err != nil {
			t.Fatalf("unexpected error: %s", err)
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
			created, err := applyTestObj(objName, "@every 5m", false, time.Time{}, tc.labels)
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
			created, err := applyTestObj(objName, "@every 5m", tc.updateStatus, tc.nextScheduledTime, nil)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
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

func applyTestObj(name, schedule string, updateStatus bool, nextScheduledTime time.Time, labels map[string]string) (*clusteropv1alpha1.KubeletUpgrade, error) {
	ctx, cancel := context.WithTimeout(context.Background(), testController.requestTimeout)
	defer cancel()

	// create the object
	obj := clusteropv1alpha1testing.NewTestKubeletUpgrade(name)
	clusteropv1alpha1testing.WithSchedule(obj, schedule)
	clusteropv1alpha1testing.WithLabels(obj, labels)
	created, err := testController.clusteropClientsets.ClusteropV1alpha1().KubeletUpgrades().Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	if !updateStatus {
		return created, nil
	}

	// update its status if necessary
	conditions := []clusteropv1alpha1.UpgradeCondition{}
	clusteropv1alpha1testing.WithStatusConditions(created, conditions)
	clusteropv1alpha1testing.WithNextScheduledTime(created, nextScheduledTime)
	return testController.clusteropClientsets.ClusteropV1alpha1().KubeletUpgrades().UpdateStatus(ctx, created, metav1.UpdateOptions{})
}
