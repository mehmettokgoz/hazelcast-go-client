package cp

import (
	"context"
	"github.com/hazelcast/hazelcast-go-client/internal/cluster"
	"github.com/hazelcast/hazelcast-go-client/internal/invocation"
	"github.com/hazelcast/hazelcast-go-client/internal/logger"
	"github.com/hazelcast/hazelcast-go-client/internal/proto/codec"
	iserialization "github.com/hazelcast/hazelcast-go-client/internal/serialization"
	"github.com/hazelcast/hazelcast-go-client/types"
	"strings"
	"sync"
	"time"
)

const (
	AtomicLongService      = "hz:raft:atomicLongService"
	AtomicReferenceService = "hz:raft:atomicRefService"
	CountDownLatchService  = "hz:raft:countDownLatchService"
	LockService            = "hz:raft:lockService"
	SemaphoreService       = "hz:raft:semaphoreService"
)

const (
	defaultGroupName    = "default"
	metadataCpGroupName = "metadata"
)

type serviceBundle struct {
	invocationService    *invocation.Service
	serializationService *iserialization.Service
	invocationFactory    *cluster.ConnectionInvocationFactory
	logger               *logger.LogAdaptor
}

func (b serviceBundle) Check() {
	if b.invocationService == nil {
		panic("invocationFactory is nil")
	}
	if b.invocationFactory == nil {
		panic("invocationService is nil")
	}
	if b.serializationService == nil {
		panic("serializationService is nil")
	}
	if b.logger == nil {
		panic("logger is nil")
	}
}

type ProxyManager struct {
	bundle  *serviceBundle
	mu      *sync.RWMutex
	proxies map[string]interface{}
}

func NewCpProxyManager(ss *iserialization.Service, cif *cluster.ConnectionInvocationFactory, is *invocation.Service, l *logger.LogAdaptor) (*ProxyManager, error) {
	b := &serviceBundle{
		invocationService:    is,
		invocationFactory:    cif,
		serializationService: ss,
		logger:               l,
	}
	b.Check()
	p := &ProxyManager{
		mu:      &sync.RWMutex{},
		proxies: map[string]interface{}{},
		bundle:  b,
	}
	return p, nil
}

func (m *ProxyManager) getOrCreateProxy(ctx context.Context, serviceName string, proxyName string, wrapProxyFn func(p *proxy) (interface{}, error)) (interface{}, error) {
	proxyName = m.withoutDefaultGroupName(ctx, proxyName)
	objectName := m.objectNameForProxy(ctx, proxyName)
	groupId, _ := m.createGroupId(ctx, proxyName)
	m.mu.RLock()
	wrapper, ok := m.proxies[proxyName]
	m.mu.RUnlock()
	if ok {
		return wrapper, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if wrapper, ok := m.proxies[proxyName]; ok {
		// someone has already created the proxy
		return wrapper, nil
	}
	p, err := newProxy(ctx, m.bundle, groupId, serviceName, proxyName, objectName)
	if err != nil {
		return nil, err
	}
	wrapper, err = wrapProxyFn(p)
	if err != nil {
		return nil, err
	}
	m.proxies[proxyName] = wrapper
	return wrapper, nil
}

func (m *ProxyManager) objectNameForProxy(ctx context.Context, name string) string {
	idx := strings.Index(name, "@")
	if idx == -1 {
		return name
	}
	groupName := strings.TrimSpace(name[idx+1:])
	if len(groupName) <= 0 {
		panic("Custom CP group name cannot be empty string")
	}
	objectName := strings.TrimSpace(name[:idx])
	if len(objectName) <= 0 {
		panic("Object name cannot be empty string")
	}
	return objectName
}

func (m *ProxyManager) createGroupId(ctx context.Context, proxyName string) (*types.RaftGroupId, error) {
	request := codec.EncodeCPGroupCreateCPGroupRequest(proxyName)
	now := time.Now()
	inv := m.bundle.invocationFactory.NewInvocationOnRandomTarget(request, nil, now)
	err := m.bundle.invocationService.SendRequest(context.Background(), inv)
	if err != nil {
		return nil, err
	}
	response, err := inv.GetWithContext(ctx)
	if err != nil {
		return nil, err
	}
	groupId := codec.DecodeCPGroupCreateCPGroupResponse(response)
	return &groupId, nil
}

func (m *ProxyManager) withoutDefaultGroupName(ctx context.Context, proxyName string) string {
	name := strings.TrimSpace(proxyName)
	idx := strings.Index(name, "@")
	if idx == -1 {
		return name
	}
	if ci := strings.Index(name[idx+1:], "@"); ci != -1 {
		panic("Custom group name must be specified at most once")
	}
	groupName := strings.TrimSpace(name[idx+1:])
	if lgn := strings.ToLower(groupName); lgn == metadataCpGroupName {
		panic("\"CP data structures cannot run on the METADATA CP group!\"")
	}
	if strings.ToLower(groupName) == defaultGroupName {
		return name[:idx]
	}
	return name
}

func (m *ProxyManager) GetAtomicLong(ctx context.Context, name string) (*AtomicLong, error) {
	p, err := m.getOrCreateProxy(ctx, AtomicLongService, name, func(p *proxy) (interface{}, error) {
		return newAtomicLong(p), nil
	})
	if err != nil {
		return nil, err
	}
	return p.(*AtomicLong), nil
}
