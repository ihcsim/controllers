package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Database represents a database
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec   `json:"spec"`
	Status DatabaseStatus `json:"status"`
}

// DatabaseSpec presents the database object spec as defined by the user.
type DatabaseSpec struct {
	Env         []corev1.EnvVar
	Image       string
	ReadOnly    bool
	Replicas    int
	Resources   corev1.ResourceRequirements
	Role        string
	TLS         bool
	Tolerations []corev1.Toleration
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DatabaseStatus represents the database object status as observed by the API
// server.
type DatabaseStatus struct {
	// Phase represents the state of the database.
	Phase string `json:"phase,omitempty"`

	// Replicas represents the current number of replicas.
	Replicas int `json:"replicas"`
}

// DatabaseList represents a list of database objects.
type DatabaseList struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Items             []Database `json:"items"`
}
