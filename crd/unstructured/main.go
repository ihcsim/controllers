package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

var (
	homeDir        = os.Getenv("HOME")
	binDir         = flag.String("bin-dir", "bin", "list of paths containing CRD YAML or JSON configs")
	crdDir         = flag.String("crd-dir", filepath.Join("testdata", "crd"), "list of comma-separated paths containing CRD YAML or JSON configs (e.g., dir1,dir2,dir3")
	resourceDir    = flag.String("resource-dir", filepath.Join("..", "testdata", "resources"), "list of comma-separated paths containing CRD YAML or JSON configs (e.g., dir1,dir2,dir3")
	requestTimeout = flag.Duration("request-timeout", time.Minute*2, "request timeout duration")
)

func init() {
	flag.Parse()
}

func main() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt)

	crdPaths := strings.Split(*crdDir, ",")
	env := envtest.Environment{
		CRDDirectoryPaths:        crdPaths,
		BinaryAssetsDirectory:    *binDir,
		AttachControlPlaneOutput: true,
		ErrorIfCRDPathMissing:    true,
	}
	defer func() {
		if err := env.Stop(); err != nil {
			log.Printf("error shutting down: %s", err)
		}
	}()

	// set up a test env with apiserver and etcd
	restConfig, err := env.Start()
	if err != nil {
		exitWithErr(err)
	}

	var (
		clientset = dynamic.NewForConfigOrDie(restConfig)
		decoder   = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	)

	// read and decode the CR resources
	resourceFiles, err := os.ReadDir(*resourceDir)
	if err != nil {
		exitWithErr(err)
	}

	fileErrs := map[string][]error{}
	collectFileErrs := func(filepath string, err error) {
		if _, exists := fileErrs[filepath]; !exists {
			fileErrs[filepath] = []error{err}
			return
		}
		fileErrs[filepath] = append(fileErrs[filepath], err)
	}

	for _, f := range resourceFiles {
		if strings.HasSuffix(f.Name(), ".swp") {
			continue
		}

		filepath := filepath.Join(*resourceDir, f.Name())
		b, err := ioutil.ReadFile(filepath)
		if err != nil {
			collectFileErrs(filepath, err)
			continue
		}

		// decode the manifest into a unstructured object
		obj := &unstructured.Unstructured{}
		_, gvk, err := decoder.Decode(b, nil, obj)
		if err != nil {
			collectFileErrs(filepath, err)
			continue
		}

		// map the GVK to the GVR and scope
		gvr, scope, err := gvrAndScope(gvk, restConfig)
		if err != nil {
			collectFileErrs(filepath, err)
			continue
		}

		var resource dynamic.ResourceInterface
		switch scope {
		case meta.RESTScopeNameNamespace:
			resource = clientset.Resource(gvr).Namespace(obj.GetNamespace())
		default:
			resource = clientset.Resource(gvr)
		}

		var (
			ctx, cancel = context.WithTimeout(context.Background(), *requestTimeout)
			opts        = metav1.CreateOptions{}
		)

		if _, err := resource.Create(ctx, obj, opts); err != nil {
			collectFileErrs(filepath, err)
			continue
		}
		cancel()
	}

	for filepath, errs := range fileErrs {
		log.Printf("[error] failed to create custom resources. file: %s, err: %s", filepath, errs)
	}

	gvr := schema.GroupVersionResource{
		Group:    "app.example.com",
		Version:  "v1alpha1",
		Resource: "databases",
	}

	var (
		ctx, cancel = context.WithTimeout(context.Background(), *requestTimeout)
		opts        = metav1.ListOptions{}
	)
	defer cancel()

	databases, err := clientset.Resource(gvr).Namespace("demo").List(ctx, opts)
	if err != nil {
		exitWithErr(err)
	}

	msg := "found custom resource databases: "
	for _, db := range databases.Items {
		msg += fmt.Sprintf("%s,", db.GetName())
	}
	log.Println(msg[:len(msg)-1])
	log.Println("use ctrl+c to exit")
	<-exit
}

func gvrAndScope(gvk *schema.GroupVersionKind, restConfig *rest.Config) (schema.GroupVersionResource, meta.RESTScopeName, error) {
	discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(restConfig)
	apiGroupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return schema.GroupVersionResource{}, "", err
	}

	mapper := restmapper.NewDiscoveryRESTMapper(apiGroupResources)
	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return schema.GroupVersionResource{}, "", err
	}

	return mapping.Resource, mapping.Scope.Name(), nil
}

func exitWithErr(err error) {
	log.Println(err)
	os.Exit(1)
}
