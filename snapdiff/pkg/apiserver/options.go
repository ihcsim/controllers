package apiserver

import (
	"io"
	"net"

	"github.com/ihcsim/controllers/snapdiff/pkg/apis/storage/v1alpha1"
	clientset "github.com/ihcsim/controllers/snapdiff/pkg/generated/clientset/versioned"
	informers "github.com/ihcsim/controllers/snapdiff/pkg/generated/informers/externalversions"
	"github.com/ihcsim/controllers/snapdiff/pkg/initializer"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apiserver/pkg/admission"
	genericapiserver "k8s.io/apiserver/pkg/server"
	genericoptions "k8s.io/apiserver/pkg/server/options"
)

const etcdPathPrefix = "/registry/storage.isim.dev"

var (
	Scheme = runtime.NewScheme()
	Codecs = serializer.NewCodecFactory(Scheme)
)

type Options struct {
	RecommendedOptions    *genericoptions.RecommendedOptions
	SharedInformerFactory informers.SharedInformerFactory
}

func NewOptions(out, eout io.Writer) *Options {
	return &Options{
		RecommendedOptions: genericoptions.NewRecommendedOptions(
			etcdPathPrefix,
			Codecs.LegacyCodec(v1alpha1.SchemeGroupVersion),
		),
	}
}

func (o *Options) Run(stopCh <-chan struct{}) error {
	recommendedConfig, err := o.RecommendedConfig()
	if err != nil {
		return err
	}

	server, err := recommendedConfig.Complete().New()
	if err != nil {
		return err
	}

	server.GenericAPIServer.AddPostStartHook("storage-apiserver-informers",
		func(context genericapiserver.PostStartHookContext) error {
			recommendedConfig.GenericConfig.SharedInformerFactory.Start(context.StopCh)
			o.SharedInformerFactory.Start(context.StopCh)
			return nil
		},
	)

	return server.GenericAPIServer.PrepareRun().Run(stopCh)
}

func (o *Options) RecommendedConfig() (*RecommendedConfig, error) {
	if err := o.RecommendedOptions.SecureServing.MaybeDefaultWithSelfSignedCerts("localhost", nil, []net.IP{net.ParseIP("127.0.0.1")}); err != nil {
		return nil, errors.Wrap(err, "failed to create self-signed certificates")
	}

	o.RecommendedOptions.ExtraAdmissionInitializers = func(c *genericapiserver.RecommendedConfig) ([]admission.PluginInitializer, error) {
		client, err := clientset.NewForConfig(c.LoopbackClientConfig)
		if err != nil {
			return nil, err
		}

		o.SharedInformerFactory = informers.NewSharedInformerFactory(client, c.LoopbackClientConfig.Timeout)
		return []admission.PluginInitializer{
			initializer.New(o.SharedInformerFactory),
		}, nil
	}

	config := genericapiserver.NewRecommendedConfig(Codecs)
	if err := o.RecommendedOptions.ApplyTo(config); err != nil {
		return nil, errors.Wrap(err, "failed to apply new configuration")
	}

	return &RecommendedConfig{
		GenericConfig: config,
	}, nil
}

func (o *Options) Validate() error {
	errors := []error{}
	errors = append(errors, o.RecommendedOptions.Validate()...)
	return utilerrors.NewAggregate(errors)
}

func (o *Options) Complete() error {
	return nil
}
