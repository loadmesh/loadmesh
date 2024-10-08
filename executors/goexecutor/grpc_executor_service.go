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

package goexecutor

import (
	"context"
	"net"

	fscommon "github.com/functionstream/function-stream/common"
	"github.com/go-logr/logr"
	"github.com/loadmesh/loadmesh/api"
	"github.com/loadmesh/loadmesh/model/protocol"
	"google.golang.org/grpc"
)

type GRPCExecutorService struct {
	executor api.Executor
	options  *options
	log      *fscommon.Logger
}

type options struct {
	lis net.Listener
	log *logr.Logger
}

type Option interface {
	apply(option *options) (*options, error)
}

type optionFunc func(*options) (*options, error)

func (f optionFunc) apply(c *options) (*options, error) {
	return f(c)
}

func WithListener(lis net.Listener) Option {
	return optionFunc(func(o *options) (*options, error) {
		o.lis = lis
		return o, nil
	})
}

func WithLogger(log *logr.Logger) Option {
	return optionFunc(func(o *options) (*options, error) {
		o.log = log
		return o, nil
	})
}

func NewGRPCExecutorService(executor api.Executor, opts ...Option) (*GRPCExecutorService, error) {
	o := &options{}
	for _, opt := range opts {
		_, err := opt.apply(o)
		if err != nil {
			return nil, err
		}
	}
	var log *fscommon.Logger
	if o.log == nil {
		log = fscommon.NewDefaultLogger()
	} else {
		log = fscommon.NewLogger(o.log)
	}
	return &GRPCExecutorService{
		executor: executor,
		options:  o,
		log:      log,
	}, nil
}

type serverImpl struct {
	protocol.UnimplementedExecutorServer
	executor api.Executor
}

func (s *serverImpl) Reconcile(ctx context.Context, resource *protocol.Resource) (*protocol.Response, error) {
	s.executor.Reconcile(ctx, resource)
	return &protocol.Response{}, nil
}

func (s *serverImpl) StatusUpdate(_ *protocol.Request, server protocol.Executor_StatusUpdateServer) error {
	for {
		select {
		case status := <-s.executor.StatusUpdate(server.Context()):
			err := server.Send(status)
			if err != nil {
				return err
			}
		case <-server.Context().Done():
			return nil
		}
	}
}

func (s *GRPCExecutorService) getServer() *grpc.Server {
	grpcSvr := grpc.NewServer()
	svrImpl := &serverImpl{
		executor: s.executor,
	}
	protocol.RegisterExecutorServer(grpcSvr, svrImpl)
	return grpcSvr
}

func (s *GRPCExecutorService) Serve(ctx context.Context) error {
	svr := s.getServer()
	go func() {
		<-ctx.Done()
		svr.GracefulStop()
	}()
	return svr.Serve(s.options.lis)
}
