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
	Offset  int64
	Size    int64
	Context []byte
	ZeroOut bool
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ChangedBlockList is a list of ChangedBlock objects.
type ChangedBlockList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ChangedBlock `json:"items"`
}
