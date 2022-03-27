package registry

import (
	"github.com/ihcsim/controllers/snapdiff/pkg/apis/storage"
	"github.com/ihcsim/controllers/snapdiff/pkg/apiserver/strategy"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/registry/generic"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
)

// REST implements a RESTStorage for API services.
type REST struct {
	*genericregistry.Store
}

// NewRest returns a RESTStorage object that will work against API services.
func NewREST(scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (*REST, error) {
	s := strategy.NewStrategy(scheme)
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &storage.ChangedBlock{} },
		NewListFunc:              func() runtime.Object { return &storage.ChangedBlockList{} },
		PredicateFunc:            strategy.MatchChangedBlock,
		DefaultQualifiedResource: storage.Resource("changedblocks"),
		CreateStrategy:           s,
		UpdateStrategy:           s,
		DeleteStrategy:           s,
		TableConvertor:           rest.NewDefaultTableConvertor(storage.Resource("changedblocks")),
	}
	options := &generic.StoreOptions{
		RESTOptions: optsGetter,
		AttrFunc:    strategy.GetAttrs,
	}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}

	return &REST{store}, nil
}

// PanicOnErr serves as a wrapper for custom registries. It panics if err is
// not nil. Otherwise, it just returns the given storage object.
func PanicOnErr(storage rest.StandardStorage, err error) rest.StandardStorage {
	if err != nil {
		err = errors.Wrap(err, "unable to create REST storage for a resource")
		panic(err)
	}
	return storage
}
