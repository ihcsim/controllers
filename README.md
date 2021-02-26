# controllers

This repository contains some sample Kubernetes controllers.

* podlister - List all the pods in a target namespace at every
  ['tick'](https://golang.org/pkg/time/#Ticker) interval. A Prometheus instance
  is set up to scrape the total request count.
