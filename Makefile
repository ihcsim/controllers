CODE_GENERATOR_FOLDER = ../vendor/k8s.io/code-generator/
CODE_GENERATOR_SCRIPT = $(CODE_GENERATOR_FOLDER)/generate-groups.sh

GO_PACKAGE_ROOT = github.com/ihcsim/controllers
GO_PACKAGE_CRD_TYPED = $(GO_PACKAGE_ROOT)/crd/typed
GO_PACKAGE_KUBELET_UPGRADE = $(GO_PACKAGE_ROOT)/kubelet-upgrade

crd/typed/pkg/generated:
	$(CODE_GENERATOR_SCRIPT) all \
		$(GO_PACKAGE_CRD_TYPED)/pkg/generated \
		$(GO_PACKAGE_CRD_TYPED)/pkg/apis \
		app.example.com:v1alpha1 \
		--go-header-file=$(CODE_GENERATOR_FOLDER)/hack/boilerplate.go.txt \
		--output-base=$${GOPATH}/src \
		-v 10

kubelet-upgrade/pkg/generated: clean-gen
	cd kubelet-upgrade && $(CODE_GENERATOR_SCRIPT) all \
		$(GO_PACKAGE_KUBELET_UPGRADE)/pkg/generated \
		$(GO_PACKAGE_KUBELET_UPGRADE)/pkg/apis \
		isim.dev:v1alpha1 \
		--go-header-file=$(CODE_GENERATOR_FOLDER)/hack/boilerplate.go.txt \
		--output-base=$${GOPATH}/src \
		-v 10

clean-gen:
	rm -rf $(GO_PACKAGE_KUBELET)/pkg/generated

testenv-bin:
	tar xvfz crd/bin/envtest-bins.tar.gz -C crd/bin --strip-components 2
