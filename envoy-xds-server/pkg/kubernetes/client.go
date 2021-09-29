package kubernetes

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tellery/saas-xds-server/pkg/utils/log"
	corev1 "k8s.io/api/core/v1"
	kubeerror "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var defaultResyncPeriod = 30 * time.Minute

type Client struct {
	logger    *logrus.Logger
	namespace string
	clientset kubernetes.Interface
	factory   informers.SharedInformerFactory
}

func NewK8sClient(kubeconfig, namespace string) (*Client, error) {
	var err error
	var config *rest.Config

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, errors.Wrapf(err, "build kubernetes config from file: %s", kubeconfig)
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, errors.Wrap(err, "build in cluster kubernetes config")
		}
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "create kubernetes client")
	}
	return &Client{
		logger:    log.Logger(),
		namespace: namespace,
		clientset: clientset,
		factory:   nil,
	}, nil
}

// GetFactory returns the instance of informers.SharedInformerFactory.
func (c *Client) GetFactory() informers.SharedInformerFactory {
	if c.factory == nil {
		c.factory = informers.NewSharedInformerFactoryWithOptions(c.clientset, defaultResyncPeriod, informers.WithNamespace(c.namespace))
	}
	return c.factory
}

// GetService returns the named service from the given namespace.
func (c *Client) GetService(name string) (*corev1.Service, bool, error) {
	service, err := c.GetFactory().Core().V1().Services().Lister().Services(c.namespace).Get(name)
	exist, err := translateNotFoundError(err)
	return service, exist, err
}

// GetAllServices returns a set of services by specify namespace.
func (c *Client) GetAllServices() ([]corev1.Service, error) {
	svcs, err := c.GetK8sClient().CoreV1().Services(c.namespace).List(context.Background(), metav1.ListOptions{})
	return svcs.Items, err
}

// GetK8sClient returns the interface of kubernetes.
func (c *Client) GetK8sClient() kubernetes.Interface {
	return c.clientset
}

func translateNotFoundError(err error) (bool, error) {
	if kubeerror.IsNotFound(err) {
		return false, nil
	}
	return err == nil, err
}
