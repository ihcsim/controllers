package client

import (
	"time"

	"github.com/ihcsim/controllers/crd/typed/pkg/generated/clientset/versioned"

	informerFactory "github.com/ihcsim/controllers/crd/typed/pkg/generated/informers/externalversions"
	informer "github.com/ihcsim/controllers/crd/typed/pkg/generated/informers/externalversions/app.example.com/v1alpha1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var (
	Clientset        kubernetes.Interface
	ClientsetCRD     versioned.Interface
	DatabaseInformer informer.DatabaseInformer

	resyncDuration = time.Minute * 10
)

// Init initializes all the clientsets using the provided rest config.
func Init(restConfig *rest.Config) (chan struct{}, error) {
	var err error
	Clientset, err = kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	ClientsetCRD, err = versioned.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	stop := make(chan struct{})
	sharedInformerFactory := informerFactory.NewSharedInformerFactory(ClientsetCRD, resyncDuration)
	DatabaseInformer = sharedInformerFactory.App().V1alpha1().Databases()

	DatabaseInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    func(new interface{}) {},
			UpdateFunc: func(old, new interface{}) {},
			DeleteFunc: func(obj interface{}) {},
		},
	)

	sharedInformerFactory.Start(stop)
	sharedInformerFactory.WaitForCacheSync(stop)

	return stop, nil
}
