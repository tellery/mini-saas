package manager

import (
	"context"
	"fmt"

	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/envoyproxy/go-control-plane/pkg/test/v3"
	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	"github.com/tellery/saas-xds-server/pkg/entity"
	"github.com/tellery/saas-xds-server/pkg/utils/log"
	"github.com/tellery/saas-xds-server/pkg/xds"
)

type ServiceManager struct {
	logger         *logrus.Logger
	cache          cache.SnapshotCache
	currentVersion int
	clientNodeId   string
	syncCh         chan *entity.UserServerSync
	UserServerMap  map[string]*entity.UserServer
}

func NewServiceManager(clientNodeId string) *ServiceManager {
	return &ServiceManager{
		logger:         log.Logger(),
		cache:          cache.NewSnapshotCache(false, cache.IDHash{}, nil),
		currentVersion: 0,
		clientNodeId:   clientNodeId,
		syncCh:         make(chan *entity.UserServerSync),
		UserServerMap:  make(map[string]*entity.UserServer),
	}
}

func (sm *ServiceManager) SyncCh() chan *entity.UserServerSync {
	return sm.syncCh
}

// actually useless
func (sm *ServiceManager) Init(userservers []*entity.UserServer) {
	sm.logger.Info("Initializing manager...")
	for _, v := range userservers {
		sm.UserServerMap[v.UserId] = v
	}
	sm.updateSnapshot(sm.values())
}

func (sm *ServiceManager) Run(stopCh <-chan struct{}) {
	sm.logger.Info("kube event handler starts running...")
	for {
		select {
		case newSync := <-sm.syncCh:
			sm.handleNewSync(newSync)
		case <-stopCh:
			return
		}
	}
}

func (sm *ServiceManager) GetServer(ctx context.Context) server.Server {
	cb := &test.Callbacks{Debug: sm.logger.GetLevel() == logrus.DebugLevel}
	srv := server.NewServer(ctx, sm.cache, cb)
	return srv
}

func (sm *ServiceManager) handleNewSync(newSync *entity.UserServerSync) {
	sm.logger.WithField("sync", newSync).Debug("Got new sync entry")
	if newSync.IsRemoved {
		sm.UserServerMap[newSync.UserId] = nil
	} else {
		oldEntry := sm.UserServerMap[newSync.UserId]
		if oldEntry == nil || !cmp.Equal(oldEntry, newSync.Entry) {
			sm.UserServerMap[newSync.UserId] = newSync.Entry
			sm.updateSnapshot(sm.values())
		}
	}
}

func (sm *ServiceManager) values() []*entity.UserServer {
	values := make([]*entity.UserServer, 0, len(sm.UserServerMap))
	for _, value := range sm.UserServerMap {
		if value != nil {
			values = append(values, value)
		}
	}
	return values
}

func (sm *ServiceManager) updateSnapshot(userServers []*entity.UserServer) error {
	sm.logger.WithField("version", sm.currentVersion).WithField("#userserver", len(userServers)).Debug("Snapshot has been updated")
	newSnapshot := xds.GenerateSnapshot(userServers, fmt.Sprintf("%d", sm.currentVersion))
	if err := newSnapshot.Consistent(); err != nil {
		return err
	}
	if err := sm.cache.SetSnapshot(sm.clientNodeId, newSnapshot); err != nil {
		return err
	}
	sm.currentVersion += 1
	return nil
}
