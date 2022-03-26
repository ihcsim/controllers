package apiserver

import (
	"k8s.io/apimachinery/pkg/version"
	genericapiserver "k8s.io/apiserver/pkg/server"
)

type RecommendedConfig struct {
	GenericConfig *genericapiserver.RecommendedConfig
}

type CompletedConfig struct {
	GenericConfig genericapiserver.CompletedConfig
}

func (c *RecommendedConfig) Complete() CompletedConfig {
	completed := CompletedConfig{
		c.GenericConfig.Complete(),
	}

	completed.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return completed
}

func (c CompletedConfig) New() (*Server, error) {
	gs, err := c.GenericConfig.New("storage-apiserver", genericapiserver.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	s := &Server{
		GenericAPIServer: gs,
	}

	// apiGroupInfo := genericapiserver.NewDefaultAPIGroupIVnfo(
	// storage.GroupName, Scheme, metav1.ParameterCodec, Codecs)
	// v1alpha1Storage := map[string]rest.Storage{}
	// v1alpha1Storage[] = customregistry.RESTInPeace()

	return s, nil
}
