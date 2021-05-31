package envtest

import (
	"strings"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var env envtest.Environment

// Start initializes a test environment with an API server and etcd. binDir
// points to a directory containing the API server and etcd binaries. If crdDir
// is non-empty, the CRDs defined in that directory will be created. To view
// the logs of API server and etcd logs (in stderr), set debug to true.
//
// The returned rest config can be used by clientsets to interact with the API
// server.
func Start(binDir, crdDir string, debug bool) (*rest.Config, error) {
	env := envtest.Environment{
		CRDDirectoryPaths:     strings.Split(crdDir, ","),
		BinaryAssetsDirectory: binDir,
		ErrorIfCRDPathMissing: true,
	}

	if debug {
		env.AttachControlPlaneOutput = true
	}

	return env.Start()
}

// Stop will terminates the test environment.
func Stop() error {
	return env.Stop()
}
