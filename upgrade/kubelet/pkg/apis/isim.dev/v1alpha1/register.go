package v1alpha1

import (
	group "github.com/ihcsim/controllers/upgrade/kubelet/pkg/apis/isim.dev"
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
	scheme.AddKnownTypes(SchemeGroupVersion, &KubeletUpgradeConfig{}, &KubeletUpgradeConfigList{})
	metav1.AddToGroupVersion(scheme, SchemeGroupVersion)
	return nil
}

// Kind takes an unqualified kind and returns back a qualified GroupKind.
func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}

// Resource takes an unqualified resource and returns a qualified GroupResource.
func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}
