package api

import "github.com/loadmesh/loadmesh/model/protocol"

type Executor interface {
	Reconcile(resource *protocol.Resource)
	StatusUpdate() <-chan *protocol.Status
}

type ExecutorSelector interface {
	Select(resource *protocol.Resource, executors map[string]Executor) (string, error)
}
