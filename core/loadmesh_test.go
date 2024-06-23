package core

import (
	"context"
	"github.com/functionstream/function-stream/common"
	"github.com/google/uuid"
	"github.com/loadmesh/loadmesh/api"
	"github.com/loadmesh/loadmesh/model/protocol"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

type TestExecutor struct {
	statusUpdateCh chan *protocol.Status
}

func (e *TestExecutor) StatusUpdate() <-chan *protocol.Status {
	return e.statusUpdateCh
}

func NewTestExecutor() api.Executor {
	return &TestExecutor{
		statusUpdateCh: make(chan *protocol.Status),
	}
}

func (e *TestExecutor) Reconcile(resource *protocol.Resource) {
	status := protocol.Status{
		Metadata: resource.Metadata,
		State:    protocol.State_RUNNING,
		Version:  resource.Version,
	}
	e.statusUpdateCh <- &status
}

func TestSingleExecutor(t *testing.T) {
	resMgr := NewMemoryResourceManager(common.NewDefaultLogger())
	executor := NewTestExecutor()
	loadMesh, err := NewLoadMesh(
		WithResourceManager(resMgr),
		WithExecutor("test", "0", executor),
	)
	assert.NoError(t, err, "Should not have an error while creating LoadMesh")
	err = loadMesh.Start(context.Background())
	assert.NoError(t, err, "Should not have an error while starting LoadMesh")

	resource := &protocol.Resource{
		Metadata: &protocol.Metadata{
			Uuid:      uuid.New().String(),
			Namespace: "default",
			Name:      "test-resource",
		},
		Kind:    "test",
		Version: 1,
		State:   protocol.State_PENDING,
	}
	err = resMgr.Set(resource)
	assert.NoError(t, err, "Should not have an error while setting resource")
	assert.Eventually(t, func() bool {
		newRes, err := resMgr.Get(resource.GetMetadata().GetUuid())
		return err == nil && newRes.State == protocol.State_RUNNING
	}, 10*time.Second, 1*time.Second)
}
