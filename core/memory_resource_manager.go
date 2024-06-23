package core

import (
	"context"
	"github.com/functionstream/function-stream/common"
	"github.com/golang/protobuf/proto"
	"github.com/loadmesh/loadmesh/api"
	"github.com/loadmesh/loadmesh/model/protocol"
	"sync"
	"time"
)

type MemoryResourceManager struct {
	sync.Mutex
	resources map[string]*protocol.Resource
	watchChs  map[string][]chan *protocol.Resource
	log       *common.Logger
}

func NewMemoryResourceManager(log *common.Logger) api.ResourceManager {
	return &MemoryResourceManager{
		resources: make(map[string]*protocol.Resource),
		watchChs:  make(map[string][]chan *protocol.Resource),
		log:       log,
	}
}

func cloneResource(resource *protocol.Resource) *protocol.Resource {
	return proto.Clone(resource).(*protocol.Resource)
}

func (m *MemoryResourceManager) Get(uuid string) (*protocol.Resource, error) {
	m.Lock()
	defer m.Unlock()
	m.log.Info("getting resource", "uuid", uuid)
	if resource, exists := m.resources[uuid]; exists {
		return cloneResource(resource), nil
	}
	return nil, nil
}

func (m *MemoryResourceManager) Set(resource *protocol.Resource) error {
	m.Lock()
	defer m.Unlock()
	m.log.Info("updating resource", "resource", resource)
	if old := m.resources[resource.GetMetadata().GetUuid()]; old != nil {
		if old.GetVersion() > resource.GetVersion() {
			m.log.Info("resource version is older than the current version", "resource", resource)
			return nil
		}
	}
	newRes := cloneResource(resource)
	newRes.Version++
	newRes.LastUpdateTime = time.Now().UnixMilli()
	m.resources[resource.GetMetadata().GetUuid()] = newRes
	// notify watchers asynchronously
	go func() {
		if chs, exists := m.watchChs[newRes.GetMetadata().GetKind()]; exists {
			for _, ch := range chs {
				ch <- cloneResource(newRes)
			}
		}
	}()
	return nil
}

func (m *MemoryResourceManager) Watch(ctx context.Context, kind string) (<-chan *protocol.Resource, error) {
	m.Lock()
	defer m.Unlock()
	ch := make(chan *protocol.Resource, 10)
	m.watchChs[kind] = append(m.watchChs[kind], ch)
	resourceSnapshot := make([]*protocol.Resource, 0, len(m.resources))
	for _, resource := range m.resources {
		if resource.GetMetadata().GetKind() == kind {
			resourceSnapshot = append(resourceSnapshot, cloneResource(resource))
		}
	}
	go func() {
		// send snapshot
		for _, resource := range resourceSnapshot {
			select {
			case ch <- cloneResource(resource):
			case <-ctx.Done():
				close(ch)
				return
			}
		}
	}()
	go func() {
		<-ctx.Done()
		m.Lock()
		defer m.Unlock()
		for i, c := range m.watchChs[kind] {
			if c == ch {
				m.watchChs[kind] = append(m.watchChs[kind][:i], m.watchChs[kind][i+1:]...)
				close(ch)
				break
			}
		}
	}()
	return ch, nil
}
