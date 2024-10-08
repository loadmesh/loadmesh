// Code generated by protoc-gen-go-grpc. DO NOT EDIT.

package protocol

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ExecutorClient is the client API for Executor service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ExecutorClient interface {
	Reconcile(ctx context.Context, in *Resource, opts ...grpc.CallOption) (*Response, error)
	StatusUpdate(ctx context.Context, in *Request, opts ...grpc.CallOption) (Executor_StatusUpdateClient, error)
}

type executorClient struct {
	cc grpc.ClientConnInterface
}

func NewExecutorClient(cc grpc.ClientConnInterface) ExecutorClient {
	return &executorClient{cc}
}

func (c *executorClient) Reconcile(ctx context.Context, in *Resource, opts ...grpc.CallOption) (*Response, error) {
	out := new(Response)
	err := c.cc.Invoke(ctx, "/loadmesh.Executor/Reconcile", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *executorClient) StatusUpdate(ctx context.Context, in *Request, opts ...grpc.CallOption) (Executor_StatusUpdateClient, error) {
	stream, err := c.cc.NewStream(ctx, &Executor_ServiceDesc.Streams[0], "/loadmesh.Executor/StatusUpdate", opts...)
	if err != nil {
		return nil, err
	}
	x := &executorStatusUpdateClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type Executor_StatusUpdateClient interface {
	Recv() (*Status, error)
	grpc.ClientStream
}

type executorStatusUpdateClient struct {
	grpc.ClientStream
}

func (x *executorStatusUpdateClient) Recv() (*Status, error) {
	m := new(Status)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ExecutorServer is the server API for Executor service.
// All implementations must embed UnimplementedExecutorServer
// for forward compatibility
type ExecutorServer interface {
	Reconcile(context.Context, *Resource) (*Response, error)
	StatusUpdate(*Request, Executor_StatusUpdateServer) error
	mustEmbedUnimplementedExecutorServer()
}

// UnimplementedExecutorServer must be embedded to have forward compatible implementations.
type UnimplementedExecutorServer struct {
}

func (UnimplementedExecutorServer) Reconcile(context.Context, *Resource) (*Response, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Reconcile not implemented")
}
func (UnimplementedExecutorServer) StatusUpdate(*Request, Executor_StatusUpdateServer) error {
	return status.Errorf(codes.Unimplemented, "method StatusUpdate not implemented")
}
func (UnimplementedExecutorServer) mustEmbedUnimplementedExecutorServer() {}

// UnsafeExecutorServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ExecutorServer will
// result in compilation errors.
type UnsafeExecutorServer interface {
	mustEmbedUnimplementedExecutorServer()
}

func RegisterExecutorServer(s grpc.ServiceRegistrar, srv ExecutorServer) {
	s.RegisterService(&Executor_ServiceDesc, srv)
}

func _Executor_Reconcile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Resource)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ExecutorServer).Reconcile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/loadmesh.Executor/Reconcile",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ExecutorServer).Reconcile(ctx, req.(*Resource))
	}
	return interceptor(ctx, in, info, handler)
}

func _Executor_StatusUpdate_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(Request)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(ExecutorServer).StatusUpdate(m, &executorStatusUpdateServer{stream})
}

type Executor_StatusUpdateServer interface {
	Send(*Status) error
	grpc.ServerStream
}

type executorStatusUpdateServer struct {
	grpc.ServerStream
}

func (x *executorStatusUpdateServer) Send(m *Status) error {
	return x.ServerStream.SendMsg(m)
}

// Executor_ServiceDesc is the grpc.ServiceDesc for Executor service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Executor_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "loadmesh.Executor",
	HandlerType: (*ExecutorServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Reconcile",
			Handler:    _Executor_Reconcile_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StatusUpdate",
			Handler:       _Executor_StatusUpdate_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "model/protocol/loadmesh.proto",
}
