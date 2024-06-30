package api

import (
	"context"
	"github.com/loadmesh/loadmesh/model/protocol"
)

type Executor interface {
	Reconcile(ctx context.Context, resource *protocol.Resource)
	StatusUpdate(ctx context.Context) <-chan *protocol.Status
}

type ExecutorSelector interface {
	Select(resource *protocol.Resource, executors map[string]Executor) (string, error)
}
