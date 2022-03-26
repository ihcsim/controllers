package install

import (
	"github.com/ihcsim/controllers/snapdiff/pkg/apis/storage"
	"github.com/ihcsim/controllers/snapdiff/pkg/apis/storage/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// Install registers the API group anda dds types to a scheme.
func Install(scheme *runtime.Scheme) {
	utilruntime.Must(storage.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	utilruntime.Must(scheme.SetVersionPriority(v1alpha1.SchemeGroupVersion))
}
