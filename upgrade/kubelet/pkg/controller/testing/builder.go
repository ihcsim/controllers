package testing

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clusteropv1alpha1 "github.com/ihcsim/controllers/upgrade/kubelet/pkg/apis/clusterop.isim.dev/v1alpha1"
)

// NewTestKubeletUpgrade returns a new instance of the KubeletUpgrade object
// with the given name.
func NewTestKubeletUpgrade(name string) *clusteropv1alpha1.KubeletUpgrade {
	return &clusteropv1alpha1.KubeletUpgrade{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}

// WithSchedule sets the schedule of the object.
func WithSchedule(obj *clusteropv1alpha1.KubeletUpgrade, schedule string) {
	obj.Spec.Schedule = schedule
}

// WithLabels sets the labels of the object.
func WithLabels(obj *clusteropv1alpha1.KubeletUpgrade, labels map[string]string) {
	if obj.Labels == nil {
		obj.Labels = map[string]string{}
	}

	for k, v := range labels {
		obj.Labels[k] = v
	}
}

// WithStatusConditions sets the status conditions of the object.
func WithStatusConditions(obj *clusteropv1alpha1.KubeletUpgrade, conditions []clusteropv1alpha1.UpgradeCondition) {
	obj.Status.Conditions = conditions
}

// WithNextScheduledTime sets the next scheduled time of the object.
func WithNextScheduledTime(obj *clusteropv1alpha1.KubeletUpgrade, nextScheduledTime time.Time) {
	obj.Status.NextScheduledTime = metav1.NewTime(nextScheduledTime)
}
