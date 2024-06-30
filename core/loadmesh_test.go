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
	"github.com/labstack/gommon/log"
	"github.com/loadmesh/loadmesh/executors/goexecutor"
	"net"
	"testing"
	"time"

	"github.com/functionstream/function-stream/common"
	"github.com/google/uuid"
	"github.com/loadmesh/loadmesh/api"
	"github.com/loadmesh/loadmesh/core/grpc_executor"
	"github.com/loadmesh/loadmesh/model/protocol"
	"github.com/stretchr/testify/assert"
)

type TestExecutor struct {
	statusUpdateCh chan *protocol.Status
}

var _ api.Executor = &TestExecutor{}

func (e *TestExecutor) StatusUpdate(_ context.Context) <-chan *protocol.Status {
	return e.statusUpdateCh
}

func NewTestExecutor() *TestExecutor {
	return &TestExecutor{
		statusUpdateCh: make(chan *protocol.Status),
	}
}

func (e *TestExecutor) Reconcile(_ context.Context, resource *protocol.Resource) {
	switch resource.GetState() {
	case protocol.State_INITIATING:
		status := protocol.Status{
			Metadata: resource.Metadata,
			State:    protocol.State_RUNNING,
			Version:  resource.Version,
		}
		e.statusUpdateCh <- &status
	case protocol.State_DELETING:
		status := protocol.Status{
			Metadata: resource.Metadata,
			State:    protocol.State_DELETED,
			Version:  resource.Version,
		}
		e.statusUpdateCh <- &status
	}
}

func TestSingleExecutor(t *testing.T) {
	executor := NewTestExecutor()
	testSingleExecutor(t, "0", executor, executor)
}

func TestGRPCExecutor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	executor := NewTestExecutor()
	lis, err := net.Listen("tcp", ":50051")
	assert.NoError(t, err, "Should not have an error while creating listener")
	svr, err := goexecutor.NewGRPCExecutorService(executor, goexecutor.WithListener(lis))
	assert.NoError(t, err, "Should not have an error while creating GRPCExecutorService")
	go func() {
		err := svr.Serve(ctx)
		assert.NoError(t, err, "Should not have an error while serving")
	}()
	endpoint := "localhost:50051"
	grpcExecutor := grpc_executor.NewGRPCExecutor(ctx, "localhost:50051", common.NewDefaultLogger())
	go grpcExecutor.Connect()
	testSingleExecutor(t, endpoint, grpcExecutor, executor)
}

func TestJavaExecutor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	endpoint := "localhost:50052"
	javaExecutor := grpc_executor.NewGRPCExecutor(ctx, "localhost:50052", common.NewDefaultLogger())
	go javaExecutor.Connect()
	testSingleExecutor(t, endpoint, javaExecutor, nil)
}

func testSingleExecutor(t *testing.T, endpoint string, executor api.Executor, testExecutor *TestExecutor) {
	resMgr := NewMemoryResourceManager(common.NewDefaultLogger())
	loadMesh, err := NewLoadMesh(
		WithResourceManager(resMgr),
		WithExecutor("test", endpoint, executor),
	)
	assert.NoError(t, err, "Should not have an error while creating LoadMesh")
	err = loadMesh.Start(context.Background())
	assert.NoError(t, err, "Should not have an error while starting LoadMesh")

	resource := &protocol.Resource{
		Metadata: &protocol.Metadata{
			Uuid:      uuid.New().String(),
			Namespace: "default",
			Name:      "test-resource",
			Kind:      "test",
		},
		Version: 1,
		State:   protocol.State_PENDING,
	}
	err = resMgr.Set(resource)
	assert.NoError(t, err, "Should not have an error while setting resource")
	var newRes *protocol.Resource
	assert.Eventually(t, func() bool {
		newRes, err = resMgr.Get(resource.GetMetadata().GetUuid())
		log.Info(newRes)
		return err == nil && newRes.State == protocol.State_RUNNING
	}, 10*time.Second, 1*time.Second)
	assert.Equal(t, endpoint, newRes.ExecutorEndpoint, "Executor endpoint should be 0")
	assert.Equal(t, resource.Metadata.String(), newRes.Metadata.String(), "Resource metadata should match")

	if testExecutor != nil {
		testExecutor.statusUpdateCh <- &protocol.Status{
			Metadata: newRes.Metadata,
			State:    protocol.State_FAILED,
			Version:  newRes.Version,
			Message:  "test error",
		}
		assert.Eventually(t, func() bool {
			newRes, err = resMgr.Get(newRes.GetMetadata().GetUuid())
			return err == nil && newRes.State == protocol.State_FAILED
		}, 10*time.Second, 1*time.Second)
		assert.Equal(t, "test error", newRes.Message, "Resource message should match")
	}

	newRes.State = protocol.State_DELETING
	err = resMgr.Set(newRes)
	assert.NoError(t, err, "Should not have an error while setting resource")
	assert.Eventually(t, func() bool {
		newRes, err = resMgr.Get(newRes.GetMetadata().GetUuid())
		return err == nil && newRes.State == protocol.State_DELETED
	}, 10*time.Second, 1*time.Second)
	assert.Equal(t, "", newRes.Message, "Resource message should be empty")
	assert.Equal(t, "", newRes.ExecutorEndpoint, "Executor endpoint should be empty")
}
