# controllers

This repository contains sample code related to Kubernetes controllers. They
are shared as-is under the Apache License v2.0.

Package            | Description
------------------ | --------------------
 `podlister`       | List all the pods in a target namespace at every ['tick'](https://golang.org/pkg/time/#Ticker) interval. A Prometheus instance is set up to scrape the total request count
`crd/dynamic`      | Sample code on how to use the [`dynamic` client](https://pkg.go.dev/k8s.io/client-go/dynamic#Interface) and [`unstructured.Unstructured` package](https://pkg.go.dev/k8s.io/apimachinery/pkg/apis/meta/v1/unstructured#Unstructured) to create and list custom resources
`crd/typed`        | Sample code on how to use the [`code-generator`](https://github.com/kubernetes/code-generator) to auto-generate the CRD clients, informers and helpers. To update these auto-generated code, run `make generate`
`upgrade/kubelet`  | (WIP) Automate the upgrades of kubelets in a cluster

The code in the `crd` package uses the
[`envtest`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/envtest)
package to set up a test environment containing an API server and etcd. To
extract the binaries, run `make testenv-bin`
