/*
 * Copyright 2024 LoadMesh Org.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package core

import (
	"context"
	"sync"
	"time"

	"github.com/functionstream/function-stream/common"
	"github.com/loadmesh/loadmesh/model/protocol"
	"google.golang.org/protobuf/proto"
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
			r.log.Info("the resource has reached retry time, retrying it now",
				"uuid", resource.GetMetadata().GetUuid(), "retry_count", resource.RetryCount,
				"wait_time_sec", waitTimeSec)
			resource.RetryCount++
			r.retryCh <- resource
			delete(r.retries, resource.GetMetadata().GetUuid())
		}
	}
}

func (r *RetryTracker) Add(resource *protocol.Resource) {
	r.Lock()
	defer r.Unlock()
	// We should make a copy of the resource to avoid unintended side effects from external modifications to the
	// original resource object.
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
