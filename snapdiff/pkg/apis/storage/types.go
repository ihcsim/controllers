package storage

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ChangedBlock is a changed block.
type ChangedBlock struct {
	metav1.TypeMeta
	metav1.ObjectMeta

	Spec ChangedBlockSpec
}

// ChangedBlockSpec is a changed block.
type ChangedBlockSpec struct {
	SnapshotBase   string
	SnapshotTarget string
	VolumeID       string
	Secrets        map[string]string
	Parameters     map[string]string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ChangedBlockList is a list of ChangedBlock objects.
type ChangedBlockList struct {
	metav1.TypeMeta
	metav1.ListMeta

	Items []ChangedBlock
}
