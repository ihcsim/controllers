package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	// SchemeGroupVersion is the grop version used to reigister objects of this
	// API.
	SchemeGroupVersion = schema.GroupVersion{
		Group:   group.GroupName,
		Version: "v1alpha1",
	}

	// SchemeBuilder has a list of functions that can add known types of this
	// API to a scheme.
	schemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

	// AddToScheme can be used to register the known types of this API with a
	// scheme
	AddToScheme = schemeBuilder.AddToScheme
)

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(SchemeGroupVersion, &KubeletUpgradeConfig{}, &KubletUpgradeConfigList{})
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}
