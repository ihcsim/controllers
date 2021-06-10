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
	FailurePolicy  string               `json:"failurePolicy"`
	MaxUnavailable int                  `json:"maxUnavailable"`
	Schedule       string               `json:"schedule"`
	Selector       metav1.LabelSelector `json:"selector"`
	Strategy       string               `json:"strategy"`
}

const (
	UpgradeStrategyRetain  = "retain"
	UpgradeStrategyReplace = "replace"

	UpgradeFailurePolicyStrict = "strict"
	UpgradeFailurePolicyIgnore = "ignore"
)

// KubeletUpgradeConfigStatus represents the status of the upgrade process.
type KubeletUpgradeConfigStatus struct {
	History           []UpgradeHistory `json:"history"`
	KubeletVersion    string           `json:"kubeletVersion"`
	LastCompletedTime *metav1.Time     `json:"lastCompletedTime"`
	LastUpgradeResult string           `json:"lastUpgradeResult"`
	NextScheduledTime *metav1.Time     `json:"nextScheduledTime"`
}

// UpgradeHistory shows the history of a kubelet upgrade.
type UpgradeHistory struct {
	KubeletVersion    string       `json:"kubeletVersion"`
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
