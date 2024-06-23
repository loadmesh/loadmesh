package core

import (
	"context"
	"fmt"
	fscommon "github.com/functionstream/function-stream/common"
	"github.com/go-logr/logr"
	"github.com/loadmesh/loadmesh/api"
	"github.com/loadmesh/loadmesh/model/protocol"
	"math/rand"
)

var (
	ErrExecutorNotReady = fmt.Errorf("executor is not ready")
)

type LoadMesh struct {
	opts      *options
	log       *fscommon.Logger
	executors map[string]*ResourceReconciler
}

type options struct {
	resourceManager   api.ResourceManager
	executorEndpoints map[string]map[string]api.Executor
	selector          api.ExecutorSelector
	log               *logr.Logger
}

type LoadMeshOption interface {
	apply(option *options) (*options, error)
}

type optionFunc func(*options) (*options, error)

func (f optionFunc) apply(c *options) (*options, error) {
	return f(c)
}

func WithResourceManager(rm api.ResourceManager) LoadMeshOption {
	return optionFunc(func(o *options) (*options, error) {
		o.resourceManager = rm
		return o, nil
	})
}

func WithExecutor(kind string, endpoint string, executor api.Executor) LoadMeshOption {
	return optionFunc(func(o *options) (*options, error) {
		if _, ok := o.executorEndpoints[kind]; !ok {
			o.executorEndpoints[kind] = make(map[string]api.Executor)
		}
		o.executorEndpoints[kind][endpoint] = executor
		return o, nil
	})
}

func WithLogger(log *logr.Logger) LoadMeshOption {
	return optionFunc(func(o *options) (*options, error) {
		o.log = log
		return o, nil
	})
}

func WithExecutorSelector(selector api.ExecutorSelector) LoadMeshOption {
	return optionFunc(func(o *options) (*options, error) {
		o.selector = selector
		return o, nil
	})
}

func NewLoadMesh(opts ...LoadMeshOption) (*LoadMesh, error) {
	o := &options{}
	o.executorEndpoints = make(map[string]map[string]api.Executor)
	for _, opt := range opts {
		_, err := opt.apply(o)
		if err != nil {
			return nil, err
		}
	}
	if o.resourceManager == nil {
		return nil, fmt.Errorf("resource manager is required")
	}
	if o.selector == nil {
		o.selector = &RandomSelector{}
	}
	var log *fscommon.Logger
	if o.log == nil {
		log = fscommon.NewDefaultLogger()
	} else {
		log = fscommon.NewLogger(o.log)
	}
	return &LoadMesh{
		opts:      o,
		log:       log,
		executors: make(map[string]*ResourceReconciler),
	}, nil
}

func (l *LoadMesh) Start(ctx context.Context) error {
	for kind, endpoints := range l.opts.executorEndpoints {
		if _, ok := l.executors[kind]; !ok {
			l.executors[kind] = &ResourceReconciler{
				kind:           kind,
				pool:           make(map[string]api.Executor),
				ctx:            ctx,
				log:            l.log,
				resMgr:         l.opts.resourceManager,
				statusUpdateCh: make(chan *protocol.Status),
				selector:       l.opts.selector,
			}
		}
		for endpoint, executor := range endpoints {
			l.executors[kind].AddExecutor(endpoint, executor)
		}
		go l.executors[kind].RunEventLoop()
	}
	return nil
}

type RandomSelector struct {
}

func (r *RandomSelector) Select(_ *protocol.Resource, executors map[string]api.Executor) (string, error) {
	if len(executors) == 0 {
		return "", ErrExecutorNotReady
	}
	endpoints := make([]string, 0, len(executors))
	for endpoint, _ := range executors {
		endpoints = append(endpoints, endpoint)
	}
	return endpoints[rand.Intn(len(endpoints))], nil
}

type ResourceReconciler struct {
	kind string
	// endpoint -> executor
	pool     map[string]api.Executor
	ctx      context.Context
	log      *fscommon.Logger
	resMgr   api.ResourceManager
	selector api.ExecutorSelector

	statusUpdateCh chan *protocol.Status
}

func (p *ResourceReconciler) AddExecutor(endpoint string, executor api.Executor) {
	p.log.Info("adding executor", "kind", p.kind, "endpoint", endpoint)
	if _, ok := p.pool[endpoint]; ok {
		p.log.Info("executor already exists", "kind", p.kind, "endpoint", endpoint)
		return
	}
	statusUpdateCh := executor.StatusUpdate()
	p.pool[endpoint] = executor
	go func() {
		for {
			select {
			case status := <-statusUpdateCh:
				p.statusUpdateCh <- status
			case <-p.ctx.Done():
				return
			}
		}
	}()
}

func (p *ResourceReconciler) RunEventLoop() {
	watchCh, err := p.resMgr.Watch(p.ctx, p.kind)
	if err != nil {
		p.log.Error(err, "failed to watch resource")
		return
	}
	p.log.Info("start event loop", "kind", p.kind)
	for {
		select {
		case resource := <-watchCh:
			p.reconcileResource(resource)
		case status := <-p.statusUpdateCh:
			p.updateStatus(status)
		case <-p.ctx.Done():
		}
	}
}

func resourceMetadataString(resource *protocol.Resource) string {
	return fmt.Sprintf("[%s]%s/%s(%s)", resource.GetKind(), resource.GetMetadata().GetNamespace(), resource.GetMetadata().GetName(), resource.GetMetadata().GetUuid())
}

func (p *ResourceReconciler) updateResource(resource *protocol.Resource) {
	if err := p.resMgr.Set(resource); err != nil {
		p.log.Error(err, "failed to update resource", "resource", resourceMetadataString(resource))
	}
}

func (p *ResourceReconciler) getResource(uuid string) *protocol.Resource {
	resource, err := p.resMgr.Get(uuid)
	if err != nil {
		p.log.Error(err, "failed to get resource", "uuid", uuid)
		return nil
	}
	return resource
}

func (p *ResourceReconciler) reconcileResource(resource *protocol.Resource) {
	switch resource.GetState() {
	case protocol.State_PENDING:
		{
			p.log.Info("assigning resource", "resource", resourceMetadataString(resource))
			resource.State = protocol.State_ASSIGNING
			p.updateResource(resource)
		}
	case protocol.State_ASSIGNING:
		{
			p.log.Info("initiating resource", "resource", resourceMetadataString(resource))
			p.assignResource(resource)
		}
	}
}

func (p *ResourceReconciler) handleResourceError(resource *protocol.Resource, err error) {
	p.log.Error(err, "failed to handle resource", "resource", resourceMetadataString(resource))
	status := &protocol.Status{}
	status.Metadata = resource.GetMetadata()
	status.State = protocol.State_FAILED
	status.Version = resource.GetVersion()
	status.Message = err.Error()
	p.statusUpdateCh <- status
}

func (p *ResourceReconciler) assignResource(resource *protocol.Resource) {
	endpoint, err := p.selector.Select(resource, p.pool)
	if err != nil {
		p.handleResourceError(resource, err)
		return
	}
	if p.pool[endpoint] == nil {
		p.handleResourceError(resource, fmt.Errorf("executor not found: %s", endpoint))
		return
	}
	resource.ExecutorEndpoint = endpoint
	resource.State = protocol.State_INITIATING
	p.updateResource(resource)
	updatedRes := p.getResource(resource.GetMetadata().GetUuid())
	p.pool[endpoint].Reconcile(updatedRes)
}

func (p *ResourceReconciler) updateStatus(status *protocol.Status) {
	resource := p.getResource(status.GetMetadata().GetUuid())
	if resource == nil {
		return
	}
	if status.GetVersion() < resource.GetVersion() {
		p.log.Info("status version is older than resource version. Ignoring", "status", status, "resource", resource)
		return
	}
	resource.State = status.GetState()
	resource.Message = status.GetMessage()
	p.updateResource(resource)
}
