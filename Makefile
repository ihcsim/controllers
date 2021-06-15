SHELL=/bin/bash

BASE_DIR = $(shell pwd)
CODE_GENERATOR_FOLDER = $(BASE_DIR)/vendor/k8s.io/code-generator
CODE_GENERATOR_SCRIPT = $(CODE_GENERATOR_FOLDER)/generate-groups.sh

GO_PACKAGE_ROOT = github.com/ihcsim/controllers
GO_PACKAGE_CRD_TYPED = $(GO_PACKAGE_ROOT)/crd/typed
GO_PACKAGE_UPGRADE_KUBELET = $(GO_PACKAGE_ROOT)/upgrade/kubelet

GROUP_VERSION_UPGRADE_KUBELET = clusterop.isim.dev:v1alpha1
GROUP_VERSION_CRD_TYPED = app.example.com:v1alpha1

generate: clean-gen
	for folder in upgrade/kubelet crd/typed; do \
		if [ "$${folder}" = "upgrade/kubelet" ]; then \
			go_package=$(GO_PACKAGE_UPGRADE_KUBELET) ;\
			group_version=$(GROUP_VERSION_UPGRADE_KUBELET) ;\
		else \
			go_package=$(GO_PACKAGE_CRD_TYPED) ;\
			group_version=$(GROUP_VERSION_CRD_TYPED) ;\
		fi &&\
		pushd $${folder} && \
		$(CODE_GENERATOR_SCRIPT) all \
			$${go_package}/pkg/generated \
			$${go_package}/pkg/apis \
			$${group_version} \
			--go-header-file=$(CODE_GENERATOR_FOLDER)/hack/boilerplate.go.txt \
			--output-base=$${GOPATH}/src && \
		popd ;\
	done

clean-gen:
	for folder in upgrade/kubelet crd/typed; do \
		rm -rf $${folder}/pkg/generated && \
		find $${folder}/pkg/apis -iname "zz_generated.deepcopy.go" | xargs rm ;\
	done

test:
	cd upgrade && go test ./...
	cd crd && go test ./...

testenv-bin:
	tar xvfz crd/bin/envtest-bins.tar.gz -C crd/bin --strip-components 2

kind:
	kind create cluster --config ./etc/kind.yaml
