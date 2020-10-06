package kubernetes

import (
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// RestConfig will be used globally across different packages
var RestConfig *rest.Config

// ClientSet will be used globally across different packages
var ClientSet kubernetes.Interface

// NewRestConfig will create a new InClusterConfig
func NewRestConfig() (*rest.Config, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return restConfig, nil
}

// NewClientSet will create a new k8s ClientSet
func NewClientSet(config *rest.Config) (kubernetes.Interface, error) {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return clientSet, nil
}
