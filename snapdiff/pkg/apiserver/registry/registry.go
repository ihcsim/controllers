package registry

import (
	"github.com/ihcsim/controllers/snapdiff/pkg/apis/storage"
	reststorage "github.com/ihcsim/controllers/snapdiff/pkg/apiserver/storage"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/rest"
)

// REST implements a RESTStorage for API services.
type REST struct {
	watch chan watch.Event
	reststorage.Memory
	rest.TableConvertor
}

// NewRest returns a RESTStorage object that will work against API services.
func NewREST() *REST {
	return &REST{
		make(chan watch.Event),
		reststorage.NewMemory(),
		rest.NewDefaultTableConvertor(storage.Resource("changedblocks")),
	}
}
