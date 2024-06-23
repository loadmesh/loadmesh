package api

import (
	"context"
	"github.com/loadmesh/loadmesh/model/protocol"
)

type ResourceManager interface {
	Get(uuid string) (*protocol.Resource, error)
	Set(resource *protocol.Resource) error
	Watch(ctx context.Context, kind string) (<-chan *protocol.Resource, error)
}
