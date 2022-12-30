package main

import gop2pdme "github.com/gop2pdme/proto"

const (
	REQUEST = iota
	ALLOWED
	DONE
	RELEASED
	WANTED
	HELD
)

type client struct {
	gop2pdme.UnimplementedP2PServiceServer
	id       int32
	state    int
	lamport  int64
	peers    map[int]gop2pdme.P2PServiceClient
	reqQueue []gop2pdme.Post
	replies  int
}
