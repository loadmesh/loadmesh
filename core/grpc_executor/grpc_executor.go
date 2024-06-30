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

package grpc_executor

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	fscommon "github.com/functionstream/function-stream/common"
	"github.com/loadmesh/loadmesh/api"
	"github.com/loadmesh/loadmesh/core/common"
	"github.com/loadmesh/loadmesh/model/protocol"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	ExecutorStatusNotReady int32 = iota
	ExecutorStatusReady    int32 = iota
)

type GRPCExecutor struct {
	endpoint       string
	client         protocol.ExecutorClient
	status         int32
	ctx            context.Context
	log            *fscommon.Logger
	statusUpdateCh chan *protocol.Status
}

var _ api.Executor = &GRPCExecutor{}

func NewGRPCExecutor(ctx context.Context, endpoint string, log *fscommon.Logger) *GRPCExecutor {
	return &GRPCExecutor{
		endpoint:       endpoint,
		ctx:            ctx,
		log:            log,
		statusUpdateCh: make(chan *protocol.Status),
	}
}

func (e *GRPCExecutor) Connect() {
	conn, err := grpc.NewClient(e.endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	handleErr := func(err error) {
		e.log.Error(err, "failed to connect to executor. Reconnecting 1 second later", "endpoint", e.endpoint)
		go func() {
			time.Sleep(1 * time.Second)
			e.Connect()
		}()
	}
	if err != nil {
		handleErr(err)
		return
	}
	e.client = protocol.NewExecutorClient(conn)
	statusUpdateClient, err := e.client.StatusUpdate(e.ctx, &protocol.Request{})
	if err != nil {
		handleErr(fmt.Errorf("failed to create status update client: %w", err))
		return
	}
	atomic.CompareAndSwapInt32(&e.status, ExecutorStatusNotReady, ExecutorStatusReady)
	e.log.Info("connected to executor", "endpoint", e.endpoint)
	for {
		status, err := statusUpdateClient.Recv()
		if err != nil {
			handleErr(fmt.Errorf("failed to receive status update: %w", err))
			return
		}
		e.log.Info("received status update", "status", status)
		e.statusUpdateCh <- status
	}
}

func (e *GRPCExecutor) handleResourceError(resource *protocol.Resource, err error) {
	status := &protocol.Status{}
	status.Metadata = resource.GetMetadata()
	status.State = resource.GetState()
	status.Version = resource.Version
	status.Message = err.Error()
	status.RetryCount = resource.GetRetryCount() + 1
	e.statusUpdateCh <- status
}

func (e *GRPCExecutor) Reconcile(ctx context.Context, resource *protocol.Resource) {
	if atomic.LoadInt32(&e.status) != ExecutorStatusReady {
		e.handleResourceError(resource, common.ErrExecutorNotReady)
		return
	}
	res, err := e.client.Reconcile(ctx, resource)
	if err != nil {
		e.handleResourceError(resource, err)
		return
	}
	if res.GetError() != "" {
		e.handleResourceError(resource, fmt.Errorf("reconcile failed: %s", res.GetError()))
		return
	}
}

func (e *GRPCExecutor) StatusUpdate(_ context.Context) <-chan *protocol.Status {
	return e.statusUpdateCh
}

type GRPCRandomSelector struct {
}

func NewGRPCRandomSelector() api.ExecutorSelector {
	return &GRPCRandomSelector{}
}

func (s *GRPCRandomSelector) Select(_ *protocol.Resource, executors map[string]api.Executor) (string, error) {
	readyExecutors := make([]string, len(executors))
	for endpoint, executor := range executors {
		if grpcExecutor, ok := executor.(*GRPCExecutor); ok {
			if atomic.LoadInt32(&grpcExecutor.status) == ExecutorStatusReady {
				readyExecutors = append(readyExecutors, endpoint)
			}
		}
	}

	if len(readyExecutors) == 0 {
		return "", fmt.Errorf("no ready executor found")
	}

	return readyExecutors[rand.Intn(len(readyExecutors))], nil
}
