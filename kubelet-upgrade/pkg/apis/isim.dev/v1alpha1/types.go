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

	Spec   KubeletUpgradeSpec   `json:"spec"`
	Status KubeletUpgradeStatus `json:"status"`
}

// KubeletUpgradeSpec represents the spec of the upgrade process
type KubeletUpgradeSpec struct{}

// KubeletUpgradeStatus represents the status of the upgrade process.
type KubeletUpgradeStatus struct{}

type KubeletUpgradeConfigList struct{}
