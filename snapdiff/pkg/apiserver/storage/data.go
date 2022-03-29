package storage

import (
	"github.com/ihcsim/controllers/snapdiff/pkg/apis/storage/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var mocks = map[string][]v1alpha1.ChangedBlock{
	"token-00": {
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
	},
	"token-01": {
		{
			metav1.TypeMeta{},
			metav1.ObjectMeta{
				Name:              "delta-03",
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
				Name:              "delta-04",
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
				Name:              "delta-05",
				Namespace:         "default",
				CreationTimestamp: metav1.Now(),
			},
			v1alpha1.ChangedBlockSpec{
				Offset: int64(1048576),
				Size:   int64(260096),
			},
		},
	},
	"token-02": {
		{
			metav1.TypeMeta{},
			metav1.ObjectMeta{
				Name:              "delta-06",
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
				Name:              "delta-07",
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
				Name:              "delta-08",
				Namespace:         "default",
				CreationTimestamp: metav1.Now(),
			},
			v1alpha1.ChangedBlockSpec{
				Offset: int64(1048576),
				Size:   int64(260096),
			},
		},
	},
}
