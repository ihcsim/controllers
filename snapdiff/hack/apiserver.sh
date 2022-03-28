#!/bin/bash

sudo /usr/local/go/bin/go run ./cmd/... \
  --etcd-servers localhost:2379 \
  --authentication-kubeconfig ~/.kube/config \
  --authorization-kubeconfig ~/.kube/config \
  --kubeconfig ~/.kube/config
