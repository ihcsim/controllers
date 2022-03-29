package registry

import (
	"context"

	"github.com/ihcsim/controllers/snapdiff/pkg/apis/storage"
	v1alpha1 "github.com/ihcsim/controllers/snapdiff/pkg/apis/storage/v1alpha1"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/apiserver/pkg/registry/rest"
)

// REST implements a RESTStorage for API services.
type REST struct {
	watch chan watch.Event
	rest.TableConvertor
}

// NewRest returns a RESTStorage object that will work against API services.
func NewREST() *REST {
	return &REST{
		make(chan watch.Event),
		rest.NewDefaultTableConvertor(storage.Resource("changedblocks")),
	}
}

// New returns an empty object that can be used with Create and Update after request data has been put into it.
// This object must be a pointer type for use with Codec.DecodeInto([]byte, runtime.Object)
func (r *REST) New() runtime.Object {
	return &storage.ChangedBlock{}
}

// NamespaceScoped returns true if the storage is namespaced
func (r *REST) NamespaceScoped() bool {
	return true
}

// Get finds a resource in the storage by name and returns it.
// Although it can return an arbitrary error value, IsNotFound(err) is true for the
// returned error value err when the specified resource is not found.
func (r *REST) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return &v1alpha1.ChangedBlock{
		metav1.TypeMeta{},
		metav1.ObjectMeta{
			Name:      "delta-00",
			Namespace: "default",
		},
		v1alpha1.ChangedBlockSpec{},
	}, nil
}

// NewList returns an empty object that can be used with the List call.
// This object must be a pointer type for use with Codec.DecodeInto([]byte, runtime.Object)
func (r *REST) NewList() runtime.Object {
	return &storage.ChangedBlockList{}
}

// List selects resources in the storage which match to the selector. 'options' can be nil.
func (r *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	objs := []v1alpha1.ChangedBlock{
		{
			metav1.TypeMeta{},
			metav1.ObjectMeta{
				Name:              "delta-00",
				Namespace:         "default",
				CreationTimestamp: metav1.Now(),
			},
			v1alpha1.ChangedBlockSpec{
				Offset: int64(1048576),
				Size:   int64(260096),
			},
		},
		{
			metav1.TypeMeta{},
			metav1.ObjectMeta{
				Name:              "delta-01",
				Namespace:         "default",
				CreationTimestamp: metav1.Now(),
			},
			v1alpha1.ChangedBlockSpec{
				Offset: int64(1048576),
				Size:   int64(260096),
			},
		},
		{
			metav1.TypeMeta{},
			metav1.ObjectMeta{
				Name:              "delta-02",
				Namespace:         "default",
				CreationTimestamp: metav1.Now(),
			},
			v1alpha1.ChangedBlockSpec{
				Offset: int64(1048576),
				Size:   int64(260096),
			},
		},
	}

	remaining := int64(10)
	return &v1alpha1.ChangedBlockList{
		metav1.TypeMeta{},
		metav1.ListMeta{
			// Continue:           "abcdefg",
			RemainingItemCount: &remaining,
		},
		objs,
	}, nil
}

// 'label' selects on labels; 'field' selects on the object's fields. Not all fields
// are supported; an error should be returned if 'field' tries to select on a field that
// isn't supported. 'resourceVersion' allows for continuing/starting a watch at a
// particular version.
func (r *REST) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	return watch.NewFake(), nil
}
