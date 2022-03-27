package strategy

import (
	"context"
	"errors"

	"github.com/ihcsim/controllers/snapdiff/pkg/apis/storage"
	"github.com/ihcsim/controllers/snapdiff/pkg/apis/storage/validation"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/generic"
	apiserverstorage "k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
)

// NewStrategy creates and retusn a ChangedBlockStrategy instance.
func NewStrategy(typer runtime.ObjectTyper) ChangedBlockStrategy {
	return ChangedBlockStrategy{typer, names.SimpleNameGenerator}
}

// GetAttrs returns labels.Set, fields.Set, the presence of Initializers if any
// and error in case the given runtime.Object is not a ChangedBlock.
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	cb, ok := obj.(*storage.ChangedBlock)
	if !ok {
		return nil, nil, errors.New("given object is not a ChangedBlock")
	}
	fields := generic.ObjectMetaFieldsSet(&cb.ObjectMeta, true)
	return labels.Set(cb.ObjectMeta.Labels), fields, nil
}

func MatchChangedBlock(label labels.Selector, field fields.Selector) apiserverstorage.SelectionPredicate {
	return apiserverstorage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

type ChangedBlockStrategy struct {
	runtime.ObjectTyper
	names.NameGenerator
}

func (ChangedBlockStrategy) NamespaceScoped() bool {
	return true
}

func (ChangedBlockStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
}

func (ChangedBlockStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
}

func (ChangedBlockStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	cb := obj.(*storage.ChangedBlock)
	return validation.ValidateChangedBlock(cb)
}

func (ChangedBlockStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	return field.ErrorList{}
}

func (ChangedBlockStrategy) AllowCreateOnUpdate() bool {
	return false
}

func (ChangedBlockStrategy) AllowUnconditionalUpdate() bool {
	return false
}

func (ChangedBlockStrategy) Canonicalize(obj runtime.Object) {
}

func (ChangedBlockStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

func (ChangedBlockStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
