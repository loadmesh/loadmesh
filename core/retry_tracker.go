package core

import (
	"context"
	"github.com/functionstream/function-stream/common"
	"github.com/golang/protobuf/proto"
	"github.com/loadmesh/loadmesh/model/protocol"
	"sync"
	"time"
)

type RetryTracker struct {
	sync.Mutex
	ctx     context.Context
	retries map[string]*protocol.Resource
	retryCh chan *protocol.Resource
	log     *common.Logger
}

func NewRetryTracker(ctx context.Context, log *common.Logger) *RetryTracker {
	return &RetryTracker{
		ctx:     ctx,
		retries: make(map[string]*protocol.Resource),
		retryCh: make(chan *protocol.Resource, 10),
		log:     log,
	}
}

func (r *RetryTracker) Start() *protocol.Resource {
	for {
		select {
		case <-time.After(1 * time.Second):
			r.retry()
		case <-r.ctx.Done():
			return nil
		}
	}
}

func min(x, y int32) int32 {
	if x < y {
		return x
	}
	return y
}

func next(current int32, max int32) int32 {
	if current > 30 {
		return max
	}
	return min(1<<current, max)
}

func (r *RetryTracker) retry() {
	r.Lock()
	defer r.Unlock()
	now := time.Now().UnixMilli()
	for _, resource := range r.retries {
		waitTimeSec := next(resource.RetryCount, 30)
		if resource.GetLastUpdateTime()+int64(waitTimeSec*1000) < now {
			r.log.Info("the resource has reached retry time, retrying it now", "uuid", resource.GetMetadata().GetUuid(), "retry_count", resource.RetryCount, "wait_time_sec", waitTimeSec)
			resource.RetryCount++
			r.retryCh <- resource
			delete(r.retries, resource.GetMetadata().GetUuid())
		}
	}
}

func (r *RetryTracker) Add(resource *protocol.Resource) {
	r.Lock()
	defer r.Unlock()
	// We should make a copy of the resource to avoid unintended side effects from external modifications to the original resource object.
	r.retries[resource.GetMetadata().GetUuid()] = proto.Clone(resource).(*protocol.Resource)
}

func (r *RetryTracker) Remove(uuid string) {
	r.Lock()
	defer r.Unlock()
	delete(r.retries, uuid)
}

func (r *RetryTracker) GetRetryCh() <-chan *protocol.Resource {
	return r.retryCh
}
