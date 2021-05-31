package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ihcsim/controllers/crd/pkg/envtest"
	"github.com/ihcsim/controllers/crd/pkg/factory"
)

var (
	binDir         = flag.String("bin-dir", "bin", "list of paths containing CRD YAML or JSON configs")
	crdDir         = flag.String("crd-dir", filepath.Join("testdata", "crd"), "list of comma-separated paths containing CRD YAML or JSON configs (e.g., dir1,dir2,dir3")
	debug          = flag.Bool("debug", false, "set to true to view logs of API server and etcd")
	resourceDir    = flag.String("resource-dir", filepath.Join("testdata", "resources"), "list of comma-separated paths containing CRD YAML or JSON configs (e.g., dir1,dir2,dir3")
	requestTimeout = flag.Duration("request-timeout", time.Minute*2, "request timeout duration")

	createFuncs = map[string]factory.CreateFunc{
		"00-namespace.yaml": factory.CreateNamespace,
		"01-pgsql.yaml":     factory.CreateDatabase,
		"02-mariadb.yaml":   factory.CreateDatabase,
	}
)

func init() {
	flag.Parse()
}

func main() {
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, os.Interrupt)

	// set up a test env with apiserver and etcd
	restConfig, err := envtest.Start(*binDir, *crdDir, *debug)
	if err != nil {
		exitWithErr(err)
	}
	defer envtest.Stop()

	if err := factory.InitClientsets(restConfig); err != nil {
		exitWithErr(err)
	}

	opts := metav1.CreateOptions{}
	if err := createResources(*resourceDir, *requestTimeout, opts); err != nil {
		exitWithErr(err)
	}
}

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

func exitWithErr(err error) {
	log.Println(err)
	os.Exit(1)
}
