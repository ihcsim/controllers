package apiserver

import (
	"github.com/ihcsim/controllers/snapdiff/pkg/apis/storage"
	"github.com/ihcsim/controllers/snapdiff/pkg/apiserver/registry"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/apiserver/pkg/registry/rest"
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

	v1alpha1Storage := map[string]rest.Storage{}
	v1alpha1Storage["changedBlocks"] = registry.NewREST()
	apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(
		storage.GroupName,
		Scheme,
		metav1.ParameterCodec,
		Codecs)
	apiGroupInfo.VersionedResourcesStorageMap["v1alpha1"] = v1alpha1Storage

	if err := s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo); err != nil {
		return nil, err
	}

	return s, nil
}
