package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ChangedBlock is a changed block.
type ChangedBlock struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ChangedBlockSpec `json:"spec"`
}

// ChangedBlockSpec is a changed block.
type ChangedBlockSpec struct {
	SnapshotBase   string            `json:"snapshotBase"`
	SnapshotTarget string            `json:"snapshotTarget"`
	VolumeID       string            `json:"volumeId"`
	Secrets        map[string]string `json:"secrets"`
	Parameters     map[string]string `json:"parameters"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ChangedBlockList is a list of ChangedBlock objects.
type ChangedBlockList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ChangedBlock `json:"items"`
}
