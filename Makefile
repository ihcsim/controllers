generate:
	cd ./crd/ && ./typed/vendor/k8s.io/code-generator/generate-groups.sh all \
		github.com/ihcsim/controllers/crd/typed/pkg/generated \
		github.com/ihcsim/controllers/crd/typed/pkg/apis \
		app.example.com:v1alpha1 \
		--go-header-file=./typed/vendor/k8s.io/code-generator/hack/boilerplate.go.txt \
		--output-base=$${GOPATH}/src \
		-v 10

testenv-bin:
	tar xvfz crd/bin/envtest-bins.tar.gz -C crd/bin --strip-components 2
