package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ihcsim/controllers/crd/pkg/client"
	"github.com/ihcsim/controllers/crd/typed/pkg/apis/app.example.com/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes/scheme"
)

var decoder = serializer.NewCodecFactory(scheme.Scheme).UniversalDecoder()

func createResources(dir string, requestTimeout time.Duration, opts metav1.CreateOptions) error {
	resourceFiles, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	createErrs := []error{}
	for _, f := range resourceFiles {
		if !strings.HasSuffix(f.Name(), ".yaml") && !strings.HasSuffix(f.Name(), ".yml") && !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		filepath := filepath.Join(dir, f.Name())
		file, err := os.Open(filepath)
		if err != nil {
			createErrs = append(createErrs, fmt.Errorf("failed to read resource. filepath: %s, error: %w", filepath, err))
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		if err := createFuncs[f.Name()](ctx, file, opts); err != nil {
			createErrs = append(createErrs, fmt.Errorf("failed to create resource. filepath: %s, error: %w", filepath, err))
			cancel()
			continue
		}
		cancel()
	}

	var errs error
	for _, e := range createErrs {
		errs = fmt.Errorf("%s\n%s", err, e)
	}

	return errs
}

// createFunc is the type of all the create functions.
type createFunc func(context.Context, io.Reader, metav1.CreateOptions) error

func createNamespace(ctx context.Context, r io.Reader, opts metav1.CreateOptions) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	obj := &corev1.Namespace{}
	if err := runtime.DecodeInto(decoder, b, obj); err != nil {
		return err
	}

	log.Printf("namespace created: %s", obj.GetName())
	if _, err := client.Clientset.CoreV1().Namespaces().Create(ctx, obj, opts); err != nil {
		return err
	}
	return nil
}

func createDatabase(ctx context.Context, r io.Reader, opts metav1.CreateOptions) error {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	obj := &v1alpha1.Database{}
	if err := runtime.DecodeInto(decoder, b, obj); err != nil {
		return err
	}

	log.Printf("database created: %s", obj.GetName())
	if _, err := client.ClientsetCRD.AppV1alpha1().Databases(obj.GetNamespace()).Create(ctx, obj, opts); err != nil {
		return err
	}
	return nil
}

func listDatabases() ([]*v1alpha1.Database, error) {
	return client.DatabaseInformer.Lister().List(labels.Everything())
}
