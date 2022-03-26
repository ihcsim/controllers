package v1alpha1

import (
	"github.com/ihcsim/controllers/snapdiff/pkg/apis/storage"
	conversion "k8s.io/apimachinery/pkg/conversion"
)

func Convert_v1alpha1_ChangedBlockList_To_storage_ChangedBlockList(
	in *ChangedBlockList, out *storage.ChangedBlockList, s conversion.Scope) error {
	return autoConvert_v1alpha1_ChangedBlockList_To_storage_ChangedBlockList(in, out, s)
}

func Convert_storage_ChangedBlockList_To_v1alpha1_ChangedBlockList(
	in *storage.ChangedBlockList, out *ChangedBlockList, s conversion.Scope) error {
	return autoConvert_storage_ChangedBlockList_To_v1alpha1_ChangedBlockList(in, out, s)
}
