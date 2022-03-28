package changedblock

import (
	"fmt"
	"io"

	"github.com/ihcsim/controllers/snapdiff/pkg/apis/storage"
	informers "github.com/ihcsim/controllers/snapdiff/pkg/generated/informers/externalversions"
	listers "github.com/ihcsim/controllers/snapdiff/pkg/generated/listers/storage/v1alpha1"
	"k8s.io/apiserver/pkg/admission"
)

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register("ChangedBlocks", func(config io.Reader) (admission.Interface, error) {
		return New()
	})
}

// New creates a new changed block admission plugin.
func New() (*changedBlockPlugin, error) {
	return &changedBlockPlugin{
		Handler: admission.NewHandler(admission.Create, admission.Update, admission.Connect),
	}, nil
}

type changedBlockPlugin struct {
	*admission.Handler
	lister listers.ChangedBlockLister
}

func (c *changedBlockPlugin) Validate(a admission.Attributes, _ admission.ObjectInterfaces) error {
	if a.GetKind().GroupKind() != storage.Kind("ChangedBlock") {
		return nil
	}

	if !c.WaitForReady() {
		return admission.NewForbidden(a, fmt.Errorf("not yet ready to handle request"))
	}

	return nil
}

func (c *changedBlockPlugin) SetStorageInformerFactory(f informers.SharedInformerFactory) {
	c.lister = f.Storage().V1alpha1().ChangedBlocks().Lister()
	c.SetReadyFunc(f.Storage().V1alpha1().ChangedBlocks().Informer().HasSynced)
}

func (c *changedBlockPlugin) ValidateInitialization() error {
	if c.lister == nil {
		return fmt.Errorf("missing policy lister")
	}
	return nil
}
