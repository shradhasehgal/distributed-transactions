// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.12
// source: distributed_transactions.proto

package protos

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

// DistributedTransactionsClient is the client API for DistributedTransactions service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DistributedTransactionsClient interface {
	BeginTransaction(ctx context.Context, in *BeginTxnPayload, opts ...grpc.CallOption) (*Reply, error)
	CommitCoordinator(ctx context.Context, in *CommitPayload, opts ...grpc.CallOption) (*Reply, error)
	CommitPeer(ctx context.Context, in *CommitPayload, opts ...grpc.CallOption) (*Reply, error)
	PerformOperationCoordinator(ctx context.Context, in *TransactionOpPayload, opts ...grpc.CallOption) (*Reply, error)
	PerformOperationPeer(ctx context.Context, in *TransactionOpPayload, opts ...grpc.CallOption) (*Reply, error)
	AbortCoordinator(ctx context.Context, in *AbortPayload, opts ...grpc.CallOption) (*Reply, error)
	AbortPeer(ctx context.Context, in *AbortPayload, opts ...grpc.CallOption) (*Reply, error)
}

type distributedTransactionsClient struct {
	cc grpc.ClientConnInterface
}

func NewDistributedTransactionsClient(cc grpc.ClientConnInterface) DistributedTransactionsClient {
	return &distributedTransactionsClient{cc}
}

func (c *distributedTransactionsClient) BeginTransaction(ctx context.Context, in *BeginTxnPayload, opts ...grpc.CallOption) (*Reply, error) {
	out := new(Reply)
	err := c.cc.Invoke(ctx, "/DistributedTransactions/beginTransaction", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedTransactionsClient) CommitCoordinator(ctx context.Context, in *CommitPayload, opts ...grpc.CallOption) (*Reply, error) {
	out := new(Reply)
	err := c.cc.Invoke(ctx, "/DistributedTransactions/commitCoordinator", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedTransactionsClient) CommitPeer(ctx context.Context, in *CommitPayload, opts ...grpc.CallOption) (*Reply, error) {
	out := new(Reply)
	err := c.cc.Invoke(ctx, "/DistributedTransactions/commitPeer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedTransactionsClient) PerformOperationCoordinator(ctx context.Context, in *TransactionOpPayload, opts ...grpc.CallOption) (*Reply, error) {
	out := new(Reply)
	err := c.cc.Invoke(ctx, "/DistributedTransactions/performOperationCoordinator", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedTransactionsClient) PerformOperationPeer(ctx context.Context, in *TransactionOpPayload, opts ...grpc.CallOption) (*Reply, error) {
	out := new(Reply)
	err := c.cc.Invoke(ctx, "/DistributedTransactions/performOperationPeer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedTransactionsClient) AbortCoordinator(ctx context.Context, in *AbortPayload, opts ...grpc.CallOption) (*Reply, error) {
	out := new(Reply)
	err := c.cc.Invoke(ctx, "/DistributedTransactions/abortCoordinator", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *distributedTransactionsClient) AbortPeer(ctx context.Context, in *AbortPayload, opts ...grpc.CallOption) (*Reply, error) {
	out := new(Reply)
	err := c.cc.Invoke(ctx, "/DistributedTransactions/abortPeer", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DistributedTransactionsServer is the server API for DistributedTransactions service.
// All implementations must embed UnimplementedDistributedTransactionsServer
// for forward compatibility
type DistributedTransactionsServer interface {
	BeginTransaction(context.Context, *BeginTxnPayload) (*Reply, error)
	CommitCoordinator(context.Context, *CommitPayload) (*Reply, error)
	CommitPeer(context.Context, *CommitPayload) (*Reply, error)
	PerformOperationCoordinator(context.Context, *TransactionOpPayload) (*Reply, error)
	PerformOperationPeer(context.Context, *TransactionOpPayload) (*Reply, error)
	AbortCoordinator(context.Context, *AbortPayload) (*Reply, error)
	AbortPeer(context.Context, *AbortPayload) (*Reply, error)
	mustEmbedUnimplementedDistributedTransactionsServer()
}

// UnimplementedDistributedTransactionsServer must be embedded to have forward compatible implementations.
type UnimplementedDistributedTransactionsServer struct {
}

func (UnimplementedDistributedTransactionsServer) BeginTransaction(context.Context, *BeginTxnPayload) (*Reply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method BeginTransaction not implemented")
}
func (UnimplementedDistributedTransactionsServer) CommitCoordinator(context.Context, *CommitPayload) (*Reply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CommitCoordinator not implemented")
}
func (UnimplementedDistributedTransactionsServer) CommitPeer(context.Context, *CommitPayload) (*Reply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CommitPeer not implemented")
}
func (UnimplementedDistributedTransactionsServer) PerformOperationCoordinator(context.Context, *TransactionOpPayload) (*Reply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PerformOperationCoordinator not implemented")
}
func (UnimplementedDistributedTransactionsServer) PerformOperationPeer(context.Context, *TransactionOpPayload) (*Reply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PerformOperationPeer not implemented")
}
func (UnimplementedDistributedTransactionsServer) AbortCoordinator(context.Context, *AbortPayload) (*Reply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AbortCoordinator not implemented")
}
func (UnimplementedDistributedTransactionsServer) AbortPeer(context.Context, *AbortPayload) (*Reply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AbortPeer not implemented")
}
func (UnimplementedDistributedTransactionsServer) mustEmbedUnimplementedDistributedTransactionsServer() {
}

// UnsafeDistributedTransactionsServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DistributedTransactionsServer will
// result in compilation errors.
type UnsafeDistributedTransactionsServer interface {
	mustEmbedUnimplementedDistributedTransactionsServer()
}

func RegisterDistributedTransactionsServer(s grpc.ServiceRegistrar, srv DistributedTransactionsServer) {
	s.RegisterService(&DistributedTransactions_ServiceDesc, srv)
}

func _DistributedTransactions_BeginTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(BeginTxnPayload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedTransactionsServer).BeginTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/DistributedTransactions/beginTransaction",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedTransactionsServer).BeginTransaction(ctx, req.(*BeginTxnPayload))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedTransactions_CommitCoordinator_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CommitPayload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedTransactionsServer).CommitCoordinator(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/DistributedTransactions/commitCoordinator",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedTransactionsServer).CommitCoordinator(ctx, req.(*CommitPayload))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedTransactions_CommitPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(CommitPayload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedTransactionsServer).CommitPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/DistributedTransactions/commitPeer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedTransactionsServer).CommitPeer(ctx, req.(*CommitPayload))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedTransactions_PerformOperationCoordinator_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TransactionOpPayload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedTransactionsServer).PerformOperationCoordinator(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/DistributedTransactions/performOperationCoordinator",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedTransactionsServer).PerformOperationCoordinator(ctx, req.(*TransactionOpPayload))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedTransactions_PerformOperationPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(TransactionOpPayload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedTransactionsServer).PerformOperationPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/DistributedTransactions/performOperationPeer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedTransactionsServer).PerformOperationPeer(ctx, req.(*TransactionOpPayload))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedTransactions_AbortCoordinator_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AbortPayload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedTransactionsServer).AbortCoordinator(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/DistributedTransactions/abortCoordinator",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedTransactionsServer).AbortCoordinator(ctx, req.(*AbortPayload))
	}
	return interceptor(ctx, in, info, handler)
}

func _DistributedTransactions_AbortPeer_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(AbortPayload)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DistributedTransactionsServer).AbortPeer(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/DistributedTransactions/abortPeer",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DistributedTransactionsServer).AbortPeer(ctx, req.(*AbortPayload))
	}
	return interceptor(ctx, in, info, handler)
}

// DistributedTransactions_ServiceDesc is the grpc.ServiceDesc for DistributedTransactions service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DistributedTransactions_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "DistributedTransactions",
	HandlerType: (*DistributedTransactionsServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "beginTransaction",
			Handler:    _DistributedTransactions_BeginTransaction_Handler,
		},
		{
			MethodName: "commitCoordinator",
			Handler:    _DistributedTransactions_CommitCoordinator_Handler,
		},
		{
			MethodName: "commitPeer",
			Handler:    _DistributedTransactions_CommitPeer_Handler,
		},
		{
			MethodName: "performOperationCoordinator",
			Handler:    _DistributedTransactions_PerformOperationCoordinator_Handler,
		},
		{
			MethodName: "performOperationPeer",
			Handler:    _DistributedTransactions_PerformOperationPeer_Handler,
		},
		{
			MethodName: "abortCoordinator",
			Handler:    _DistributedTransactions_AbortCoordinator_Handler,
		},
		{
			MethodName: "abortPeer",
			Handler:    _DistributedTransactions_AbortPeer_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "distributed_transactions.proto",
}
