package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeletUpgradeConfig defines configuration that manages the kubelet upgrade
// process.
type KubeletUpgradeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KubeletUpgradeConfigSpec   `json:"spec"`
	Status KubeletUpgradeConfigStatus `json:"status"`
}

// KubeletUpgradeConfigSpec represents the spec of the upgrade process
type KubeletUpgradeConfigSpec struct {
	FailurePolicy  UpgradeFailurePolicy `json:"failurePolicy"`
	Schedule       UpgradeSchedule      `json:"schedule"`
	Selector       metav1.LabelSelector `json:"selector"`
	Strategy       UpgradeStrategy      `json:"strategy"`
	MaxUnavailable int                  `json:"maxUnavailable"`
}

// UpgradeSchedule defines the schedule in cron format.
type UpgradeSchedule string

// UpgradeStrategy defines the strategies to execute the upgrade i.e. node
// replacement or inplace upgrade.
type UpgradeStrategy int

const (
	UpgradeStrategyInplace UpgradeStrategy = iota
	UpgradeStrategyReplace
)

// UpgradeFailurePolicy defines the policy to deal with upgrade failures i.e.
// terminate process or process to next node.
type UpgradeFailurePolicy int

const (
	UpgradeFailurePolicyHalt UpgradeFailurePolicy = iota
	UpgradeFailurePolicyIgnore
)

// KubeletUpgradeConfigStatus represents the status of the upgrade process.
type KubeletUpgradeConfigStatus struct {
	History []UpgradeHistory `json:"history"`
}

// UpgradeHistory shows the history of a kubelet upgrade.
type UpgradeHistory struct {
	LastScheduledTime *metav1.Time `json:"lastScheduledTime"`
	LastCompletedTime *metav1.Time `json:"lastCompletedTime"`
	Node              string       `json:"node"`
	Outcome           string       `json:"outcome"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeletUpgradeConfigList represents a list of kubelet upgrade config
// objects.
type KubeletUpgradeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeletUpgradeConfig `json:"items"`
}
