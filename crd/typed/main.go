package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ihcsim/controllers/crd/pkg/client"
	"github.com/ihcsim/controllers/crd/pkg/envtest"
)

var (
	binDir         = flag.String("bin-dir", "bin", "list of paths containing CRD YAML or JSON configs")
	crdDir         = flag.String("crd-dir", filepath.Join("testdata", "crd"), "list of comma-separated paths containing CRD YAML or JSON configs (e.g., dir1,dir2,dir3")
	debug          = flag.Bool("debug", false, "set to true to view logs of API server and etcd")
	resourceDir    = flag.String("resource-dir", filepath.Join("testdata", "resources"), "list of comma-separated paths containing CRD YAML or JSON configs (e.g., dir1,dir2,dir3")
	requestTimeout = flag.Duration("request-timeout", time.Minute*2, "request timeout duration")

	createFuncs = map[string]createFunc{
		"00-namespace.yaml": createNamespace,
		"01-pgsql.yaml":     createDatabase,
		"02-mariadb.yaml":   createDatabase,
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

	if err := client.Init(restConfig); err != nil {
		exitWithErr(err)
	}

	opts := metav1.CreateOptions{}
	if err := createResources(*resourceDir, *requestTimeout, opts); err != nil {
		exitWithErr(err)
	}

	time.Sleep(time.Second * 1)
	databases, err := listDatabases()
	if err != nil {
		exitWithErr(err)
	}

	msg := "found custom resource databases: "
	for _, db := range databases {
		msg += fmt.Sprintf("%s,", db.GetName())
	}
	log.Println(msg[:len(msg)-1])
	log.Println("use ctrl+c to exit")
	<-exit
}

func exitWithErr(err error) {
	log.Println(err)
	os.Exit(1)
}
