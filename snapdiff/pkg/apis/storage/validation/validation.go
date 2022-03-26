package validation

import (
	"github.com/ihcsim/controllers/snapdiff/pkg/apis/storage"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

func ValidateChangedBlock(o *storage.ChangedBlock) field.ErrorList {
	return ValidateChangedBlockSpec(&o.Spec, field.NewPath("spec"))
}

func ValidateChangedBlockSpec(o *storage.ChangedBlockSpec, f *field.Path) field.ErrorList {
	errs := field.ErrorList{}

	if o.SnapshotBase != "" {
		errs = append(errs, field.Invalid(
			f.Child("snapshotBase"),
			o.SnapshotBase,
			"cannot be empty",
		))
	}

	if o.SnapshotTarget != "" {
		errs = append(errs, field.Invalid(
			f.Child("snapshotTarget"),
			o.SnapshotTarget,
			"cannot be empty",
		))
	}

	return errs
}
