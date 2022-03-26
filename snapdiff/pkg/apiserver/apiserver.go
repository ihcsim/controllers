package apiserver

import (
	genericapiserver "k8s.io/apiserver/pkg/server"
)

// Server contains the state for a Kubernetes custom api server.
type Server struct {
	GenericAPIServer *genericapiserver.GenericAPIServer
}
