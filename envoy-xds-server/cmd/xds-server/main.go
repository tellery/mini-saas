package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/oklog/run"
	"github.com/sirupsen/logrus"
	"github.com/tellery/saas-xds-server/pkg/kubernetes"
	"github.com/tellery/saas-xds-server/pkg/manager"
	"github.com/tellery/saas-xds-server/pkg/utils/log"
	"github.com/tellery/saas-xds-server/pkg/xds"
)

var (
	logger = log.Logger()

	isDebug bool

	port uint

	nodeId string

	kubeconfig string

	serverClusterName string

	namespace string
)

func init() {
	flag.BoolVar(&isDebug, "debug", false, "Enable xDS server debug logging")

	// The port that this xDS server listens on
	flag.UintVar(&port, "port", 18000, "xDS management server port")

	flag.StringVar(&nodeId, "nodeId", "test-id", "client Node ID")

	flag.StringVar(&serverClusterName, "serverClusterName", "xds_cluster", "server cluster name")

	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file.")

	flag.StringVar(&namespace, "namespace", "tellery-saas-demo", "namespace that tellery services stay")

	flag.Parse()

	if isDebug {
		logger.SetLevel(logrus.DebugLevel)
	}
}

func main() {

	logger.Debug("Debug on")

	// init service manager
	serviceManager := manager.NewServiceManager(nodeId)

	// initialize kubernetes controller
	kClient, err := kubernetes.NewK8sClient(kubeconfig, namespace)
	if err != nil {
		panic(err)
	}

	kubeController := kubernetes.NewController(kClient, serviceManager.SyncCh())

	ctxServer, cancelServer := context.WithCancel(context.Background())

	ctxController, cancelController := context.WithCancel(context.Background())

	ctxManager, cancelManager := context.WithCancel(context.Background())

	// initialize xds server
	server := serviceManager.GetServer(ctxServer)

	// initialize service manager
	// Acutally the kube informer will load all services as Add event in the very beginning
	// But this will lead the version number to increase rapidly
	// To avoid client retrieving configs multiple times, we initialize it right here.
	currentUserServers, err := kubeController.LoadCurrentSvcs()
	if err != nil {
		panic(err)
	}
	serviceManager.Init(currentUserServers)

	var g run.Group

	{
		g.Add(func() error {
			xds.RunServer(server, port, ctxServer.Done())
			logger.Info("xds server stopped")
			return nil
		}, func(err error) {
			logger.Warn("Stopping xds server")
			cancelServer()
		})
	}
	{
		g.Add(func() error {
			kubeController.Run(ctxController.Done())
			logger.Info("kubernetes controller stopped")
			return nil
		}, func(err error) {
			logger.Warn("Stopping kubernetes controller")
			cancelController()
		})
	}
	{
		g.Add(func() error {
			serviceManager.Run(ctxManager.Done())
			logger.Info("service manager stopped")
			return nil
		}, func(err error) {
			logger.Warn("Stopping service manager")
			cancelManager()
		})
	}
	{
		sigterm := make(chan os.Signal, 1)
		signal.Notify(sigterm, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP, syscall.SIGQUIT)
		cancel := make(chan struct{})
		g.Add(func() error {
			for {
				select {
				case <-sigterm:
					return nil
				case <-cancel:
					return nil
				}
			}
		}, func(err error) {
			close(cancel)
		})
	}
	if err := g.Run(); err != nil {
		logger.WithError(err).Fatal("Error occurs")
	}
}
