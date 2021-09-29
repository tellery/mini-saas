package kubernetes

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/tellery/saas-xds-server/pkg/constant"
	"github.com/tellery/saas-xds-server/pkg/entity"
	"github.com/tellery/saas-xds-server/pkg/utils/log"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
)

type ServiceListenerController struct {
	logger *logrus.Logger
	client *Client
	syncCh chan *entity.UserServerSync
}

func NewController(
	client *Client,
	syncCh chan *entity.UserServerSync,
) *ServiceListenerController {

	controller := &ServiceListenerController{
		logger: log.Logger(),
		client: client,
		syncCh: syncCh,
	}
	return controller
}

func (slc *ServiceListenerController) Run(stopCh <-chan struct{}) {
	defer utilruntime.HandleCrash()

	slc.logger.Info("kube controller is running...")

	slc.client.GetFactory().Core().V1().Services().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    slc.onAdd,
		UpdateFunc: slc.onUpdate,
		DeleteFunc: slc.onDelete,
	})
	slc.client.GetFactory().Start(stopCh)
	slc.logger.Debug("Waiting for informer caches to sync")
	for t, ok := range slc.client.GetFactory().WaitForCacheSync(stopCh) {
		if !ok {
			slc.logger.Error("timed out waiting for controller caches to sync", t.String())
			return
		}
	}
	<-stopCh
}

func (slc *ServiceListenerController) LoadCurrentSvcs() ([]*entity.UserServer, error) {
	svcs, err := slc.client.GetAllServices()
	userservers := make([]*entity.UserServer, 0, len(svcs))
	if err != nil {
		return userservers, err
	}
	for _, svc := range svcs {
		newUserServer := svcToUserServer(&svc)
		if newUserServer != nil {
			userservers = append(userservers, newUserServer)
		}
	}
	slc.logger.WithField("services", svcs).Debug("Loaded current services")
	return userservers, nil
}

func svcToUserServer(svc *corev1.Service) *entity.UserServer {
	userId := svc.Annotations[constant.UserIdAnnotation]
	if userId == "" {
		return nil
	}
	return &entity.UserServer{
		UserId:      userId,
		ServiceName: fmt.Sprintf("%s.dev.svc.cluster.local", svc.Name),
		ServicePort: uint32(svc.Spec.Ports[0].Port),
	}
}

func svcToUserServerSync(svc *corev1.Service, isRemoved bool) *entity.UserServerSync {
	entry := svcToUserServer(svc)
	if entry == nil {
		return nil
	}
	return &entity.UserServerSync{
		UserId:    entry.UserId,
		IsRemoved: isRemoved,
		Entry:     entry,
	}
}

func (slc *ServiceListenerController) onAdd(obj interface{}) {
	svc, err := convertToService(obj)
	if err != nil {
		slc.logger.WithError(err).Error("convert to service failed")
		return
	}
	slc.logger.WithField("service", svc.Name).Debug("Service added")
	userServer := svcToUserServerSync(svc, false)
	if userServer != nil {
		slc.syncCh <- userServer
	}
}

func (slc *ServiceListenerController) onUpdate(old, new interface{}) {
	newSvc, err := convertToService(new)
	if err != nil {
		slc.logger.WithError(err).Error("convert to service failed")
		return
	}
	oldSvc, err := convertToService(old)
	if err != nil {
		slc.logger.WithError(err).Error("convert to service failed")
		return
	}
	if newSvc.ResourceVersion != oldSvc.ResourceVersion {
		slc.logger.WithField("service", oldSvc.Name).Debug("Service updated")
		userServer := svcToUserServerSync(newSvc, false)
		if userServer != nil {
			slc.syncCh <- userServer
		}
	}
}

func (slc *ServiceListenerController) onDelete(obj interface{}) {
	svc, err := convertToService(obj)
	if err != nil {
		slc.logger.WithError(err).Error("convert to service failed")
		return
	}
	slc.logger.WithField("service", svc.Name).Debug("Service deleted")
	userServer := svcToUserServerSync(svc, true)
	if userServer != nil {
		slc.syncCh <- userServer
	}
}

func convertToService(o interface{}) (*corev1.Service, error) {
	service, ok := o.(*corev1.Service)
	if ok {
		return service, nil
	}

	deletedState, ok := o.(cache.DeletedFinalStateUnknown)
	if !ok {
		return nil, errors.Errorf("received unexpected object: %v", o)
	}
	service, ok = deletedState.Obj.(*corev1.Service)
	if !ok {
		return nil, errors.Errorf("deletedFinalStateUnknown contained non-Service object: %v", deletedState.Obj)
	}
	return service, nil
}
