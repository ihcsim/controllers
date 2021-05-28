# controllers

This repository contains some sample Kubernetes controllers.

* podlister - List all the pods in a target namespace at every
  ['tick'](https://golang.org/pkg/time/#Ticker) interval. A Prometheus instance
  is set up to scrape the total request count.
* crd - Examples on working with CRDs
  * unstructured - Sample code on how to use the
    [`dynamic` client](https://pkg.go.dev/k8s.io/client-go/dynamic#Interface) and
    [`unstructured.Unstructured` package](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured#Unstructured)
    to create and list custom resources.

The `crd` examples use the
[`envtest`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest)
package to set up a test environment with an API server and etcd. To extract
the binaries, run:

```sh
make testenv-bin
```
