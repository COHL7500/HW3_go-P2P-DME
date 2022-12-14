// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.6.1
// source: proto/gop2pdme.proto

package gop2pdme

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

// P2PServiceClient is the client API for P2PService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type P2PServiceClient interface {
	Recv(ctx context.Context, in *Post, opts ...grpc.CallOption) (*Empty, error)
}

type p2PServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewP2PServiceClient(cc grpc.ClientConnInterface) P2PServiceClient {
	return &p2PServiceClient{cc}
}

func (c *p2PServiceClient) Recv(ctx context.Context, in *Post, opts ...grpc.CallOption) (*Empty, error) {
	out := new(Empty)
	err := c.cc.Invoke(ctx, "/gop2pdme.P2PService/Recv", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// P2PServiceServer is the server API for P2PService service.
// All implementations must embed UnimplementedP2PServiceServer
// for forward compatibility
type P2PServiceServer interface {
	Recv(context.Context, *Post) (*Empty, error)
	mustEmbedUnimplementedP2PServiceServer()
}

// UnimplementedP2PServiceServer must be embedded to have forward compatible implementations.
type UnimplementedP2PServiceServer struct {
}

func (UnimplementedP2PServiceServer) Recv(context.Context, *Post) (*Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Recv not implemented")
}
func (UnimplementedP2PServiceServer) mustEmbedUnimplementedP2PServiceServer() {}

// UnsafeP2PServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to P2PServiceServer will
// result in compilation errors.
type UnsafeP2PServiceServer interface {
	mustEmbedUnimplementedP2PServiceServer()
}

func RegisterP2PServiceServer(s grpc.ServiceRegistrar, srv P2PServiceServer) {
	s.RegisterService(&P2PService_ServiceDesc, srv)
}

func _P2PService_Recv_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Post)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(P2PServiceServer).Recv(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/gop2pdme.P2PService/Recv",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(P2PServiceServer).Recv(ctx, req.(*Post))
	}
	return interceptor(ctx, in, info, handler)
}

// P2PService_ServiceDesc is the grpc.ServiceDesc for P2PService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var P2PService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "gop2pdme.P2PService",
	HandlerType: (*P2PServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Recv",
			Handler:    _P2PService_Recv_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/gop2pdme.proto",
}
