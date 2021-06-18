package v1alpha1

import (
	"fmt"
	"time"

	"github.com/ihcsim/controllers/upgrade/kubelet/pkg/controller/labels"
	"github.com/robfig/cron"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeletUpgrade defines configuration that manages the kubelet upgrade
// process.
type KubeletUpgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubeletUpgradeSpec   `json:"spec"`
	Status KubeletUpgradeStatus `json:"status"`
}

// KubeletUpgradeSpec represents the spec of the upgrade process
type KubeletUpgradeSpec struct {
	FailurePolicy        string               `json:"failurePolicy"`
	MaxUnavailable       int                  `json:"maxUnavailable"`
	Schedule             string               `json:"schedule"`
	Selector             metav1.LabelSelector `json:"selector"`
	Strategy             string               `json:"strategy"`
	TargetKubeletVersion string               `json:"targetKubeletVersion"`
}

const (
	ConditionMessageUpdateNextScheduleTime = "updated next schedule time to %s"
	ConditionMessageUpgradeStarted         = "started kubelet upgrade"
	ConditionMessageUpgradeCompleted       = "completed kubelet upgrade"

	ConditionReasonNextScheduledTimeStale = "next scheduled time was stale"

	ConditionStatusStarted   = "started"
	ConditionStatusCompleted = "completed"
	ConditionStatusFailed    = "failed"

	ConditionTypeUpdatedNextScheduledTime = "UpdatedNextScheduledTime"
	ConditionTypeUpgradeStarted           = "UpgradeStarted"
	ConditionTypeUpgradeCompleted         = "UpgradeCompleted"

	UpgradeFailurePolicyStrict = "strict"
	UpgradeFailurePolicyIgnore = "ignore"

	UpgradeStrategyRetain  = "retain"
	UpgradeStrategyReplace = "replace"
)

// KubeletUpgradeStatus represents the status of the upgrade process.
type KubeletUpgradeStatus struct {
	Conditions        []UpgradeCondition `json:"conditions"`
	KubeletVersion    string             `json:"kubeletVersion"`
	NextScheduledTime metav1.Time        `json:"nextScheduledTime"`
}

// UpgradeCondition shows the observed condition of an upgrade.
type UpgradeCondition struct {
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	Message            string      `json:"message"`
	Reason             string      `json:"reason"`
	Status             string      `json:"status"`
	Type               string      `json:"type"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeletUpgradeList represents a list of kubelet upgrade config
// objects.
type KubeletUpgradeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeletUpgrade `json:"items"`
}

// UpdateNextScheduledTime makes a clone of the KubeletUpgrade object and
// updates its next scheduled time.
func (k KubeletUpgrade) UpdateNextScheduledTime(now time.Time) *KubeletUpgrade {
	cloned := k.DeepCopy()
	condition := UpgradeCondition{
		LastTransitionTime: metav1.NewTime(now),
		Type:               ConditionTypeUpdatedNextScheduledTime,
	}

	schedule := cloned.Spec.Schedule
	cronSpec, err := cron.Parse(schedule)
	if err != nil {
		condition.Status = ConditionStatusFailed
		condition.Reason = err.Error()
		return cloned
	}

	nextScheduledTime := metav1.NewTime(cronSpec.Next(now).UTC())
	cloned.Status.NextScheduledTime = nextScheduledTime

	condition.Message = fmt.Sprintf(ConditionMessageUpdateNextScheduleTime, nextScheduledTime.UTC())
	condition.Reason = ConditionReasonNextScheduledTimeStale
	condition.Status = ConditionStatusCompleted
	cloned.Status.Conditions = append(cloned.Status.Conditions, condition)

	return cloned
}

// RecordUpgradeStarted makes a clone of the KubeletUpgrade and updates its
// status with an "upgrade started" condition.
func (k KubeletUpgrade) RecordUpgradeStarted(now time.Time) (*KubeletUpgrade, error) {
	cloned := k.DeepCopy()

	// add new label to show upgrade has started
	requirements, err := labels.ExcludedKubeletUpgrade()
	if err != nil {
		return nil, err
	}

	if cloned.Labels == nil {
		cloned.Labels = map[string]string{}
	}

	for _, requirement := range requirements {
		cloned.Labels[requirement.Key()] = "true"
	}

	// add new condition
	condition := UpgradeCondition{
		LastTransitionTime: metav1.NewTime(now),
		Type:               ConditionTypeUpgradeStarted,
	}

	condition.Message = ConditionMessageUpgradeStarted
	condition.Status = ConditionStatusStarted
	cloned.Status.Conditions = append(cloned.Status.Conditions, condition)

	return cloned, nil
}

// RecordUpgradeStarted makes a clone of the KubeletUpgrade and updates its
// status with an "upgrade completed" condition.
func (k KubeletUpgrade) RecordUpgradeCompleted(err error, now time.Time) (*KubeletUpgrade, error) {
	cloned := k.DeepCopy()
	condition := UpgradeCondition{
		LastTransitionTime: metav1.NewTime(now),
		Type:               ConditionTypeUpgradeCompleted,
	}

	// remove 'skip' label so that it will be processed in the next cycle
	if cloned.Labels != nil {
		requirements, err := labels.ExcludedKubeletUpgrade()
		if err != nil {
			return nil, err
		}

		klog.V(4).Infof("updating %s by removing labels: %s", cloned.GetName(), requirements)
		for _, requirement := range requirements {
			delete(cloned.Labels, requirement.Key())
		}
	}

	condition.Message = ConditionMessageUpgradeCompleted
	condition.Status = ConditionStatusCompleted

	if err != nil {
		condition.Reason = fmt.Sprintf("%s", err)
		condition.Status = ConditionStatusFailed
	}

	cloned.Status.Conditions = append(cloned.Status.Conditions, condition)
	return cloned, nil
}
