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
type KubeletUpgradeConfigSpec struct{}

// KubeletUpgradeConfigStatus represents the status of the upgrade process.
type KubeletUpgradeConfigStatus struct{}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KubeletUpgradeConfigList represents a list of kubelet upgrade config
// objects.
type KubeletUpgradeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KubeletUpgradeConfig `json:"items"`
}
