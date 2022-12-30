package main

import (
	"context"
	p2pdme "github.com/gop2pdme/proto"
	"google.golang.org/grpc"
	"log"
)

// This go file defines a peer.

func (c *client) Send(id int, request int) {
	c.lamport++
	msg := p2pdme.Post{Id: c.id, Request: int32(request), Lamport: c.lamport}
	log.Printf("Sent message %v to %v (lamport %v)", request, id, c.lamport)

	if _, err := c.peers[id].Recv(context.Background(), &msg, grpc.WaitForReady(true)); err != nil {
		log.Fatalf("Failed to send %v a message: %v, id, err", id, err)
	}
}

func (c *client) Broadcast(request int) {
	for i, _ := range c.peers {
		c.Send(i, request)
	}
}

func (c *client) Recv(_ context.Context, response *p2pdme.Post) (*p2pdme.Empty, error) {
	if response.Lamport > c.lamport {
		c.lamport = response.Lamport + 1
	} else {
		c.lamport++
	}

	log.Printf("Received message %v by %d (lamport %v", response.Request, response.Id, c.lamport)

	if response.Request == REQUEST {
		if c.state == HELD || (c.state == WANTED && c.id < response.Id) {
			c.reqQueue = append(c.reqQueue, *response)
		} else {
			c.Send(int(response.Id), ALLOWED)
		}
	} else {
		c.replies++
	}

	return &p2pdme.Empty{}, nil
}
