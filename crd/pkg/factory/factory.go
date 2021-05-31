package factory

import (
	"context"
	"io"
	"io/ioutil"

	v1alpha1 "github.com/ihcsim/controllers/crd/typed/pkg/apis/app.example.com/v1alpha1"
	"github.com/ihcsim/controllers/crd/typed/pkg/generated/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

var (
	decoder = serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder()

	clientset    kubernetes.Interface
	clientsetCRD versioned.Interface
)

// CreateFunc is the type of all the create functions.
type CreateFunc func(context.Context, io.Reader, metav1.CreateOptions) error

// InitClientsets initializes all the clientsets using the provided rest config.
func InitClientsets(restConfig *rest.Config) error {
	var err error
	clientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	clientsetCRD, err = versioned.NewForConfig(restConfig)
	if err != nil {
		return err
	}

	return nil
}

// CreateNamespace tries to decode the provided data into a namespace object.
func CreateNamespace(ctx context.Context, r io.Reader, opts metav1.CreateOptions) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	obj := &corev1.Namespace{}
	if err := runtime.DecodeInto(decoder, b, obj); err != nil {
		return err
	}

	if _, err := clientset.CoreV1().Namespaces().Create(ctx, obj, opts); err != nil {
		return err
	}
	return nil
}

// CreateDatabase tries to create a database object from the provided data.
func CreateDatabase(ctx context.Context, r io.Reader, opts metav1.CreateOptions) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	obj := &v1alpha1.Database{}
	if err := runtime.DecodeInto(decoder, b, obj); err != nil {
		return err
	}

	if _, err := clientsetCRD.AppV1alpha1().Databases(obj.GetNamespace()).Create(ctx, obj, opts); err != nil {
		return err
	}
	return nil
}
