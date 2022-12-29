package main

import (
	gop2pdme "github.com/gop2pdme/proto"
	"google.golang.org/grpc"
)

const (
	RELEASED = iota
	WANTED
	HELD
)

var (
	id         int32 = -1
	state            = RELEASED
	lamport    int64 = 0
	peersCount int32
	peers      []peer
	server     *grpc.Server
)

type peer struct {
	id      int32
	chanIn  chan gop2pdme.Post
	chanOut chan gop2pdme.Post
}

type service struct {
	gop2pdme.UnimplementedP2PServiceServer
}
