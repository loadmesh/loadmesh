package core

import (
	"context"
	"github.com/functionstream/function-stream/common"
	"github.com/google/uuid"
	"github.com/loadmesh/loadmesh/api"
	"github.com/loadmesh/loadmesh/core/grpc_executor"
	"github.com/loadmesh/loadmesh/model/protocol"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
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
	svr, err := grpc_executor.NewGRPCExecutorService(executor, grpc_executor.WithListener(lis))
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
		return err == nil && newRes.State == protocol.State_RUNNING
	}, 10*time.Second, 1*time.Second)
	assert.Equal(t, endpoint, newRes.ExecutorEndpoint, "Executor endpoint should be 0")
	assert.Equal(t, resource.Metadata.String(), newRes.Metadata.String(), "Resource metadata should match")

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
