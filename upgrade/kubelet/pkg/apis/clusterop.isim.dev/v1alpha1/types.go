package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	UpgradeStrategyRetain  = "retain"
	UpgradeStrategyReplace = "replace"

	UpgradeFailurePolicyStrict = "strict"
	UpgradeFailurePolicyIgnore = "ignore"
)

// KubeletUpgradeStatus represents the status of the upgrade process.
type KubeletUpgradeStatus struct {
	Conditions        []UpgradeCondition `json:"conditions"`
	KubeletVersion    string             `json:"kubeletVersion"`
	NextScheduledTime metav1.Time        `json:"nextScheduledTime"`
}

// UpgradeCondition shows the observed condition of an upgrade.
type UpgradeCondition struct {
	LastTransitionTime metav1.Time
	Message            string
	Reason             string
	Status             string
	Type               string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeletUpgradeList represents a list of kubelet upgrade config
// objects.
type KubeletUpgradeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeletUpgrade `json:"items"`
}
