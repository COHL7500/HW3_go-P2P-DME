package main

import (
	"context"
	"fmt"
	gop2pdme "github.com/gop2pdme/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

func (c *client) Send(id int, request int) {
	c.lamport++
	msg := gop2pdme.Post{Id: c.id, Request: int32(request), Lamport: c.lamport}
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

func (c *client) Recv(_ context.Context, response *gop2pdme.Post) (*gop2pdme.Empty, error) {
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

	return &gop2pdme.Empty{}, nil
}

func main() {
	peerId, _ := strconv.ParseInt(os.Args[1], 10, 32)
	peerCount, _ := strconv.ParseInt(os.Args[2], 10, 32)

	c := &client{
		id:       int32(peerId),
		state:    RELEASED,
		lamport:  0,
		peers:    make(map[int]gop2pdme.P2PServiceClient, int(peerCount)),
		reqQueue: make([]gop2pdme.Post, 0),
		replies:  0,
	}

	listen, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", int(c.id)+5000))

	if err != nil {
		log.Fatalf("Failed to listen %v", listen.Addr())
	}

	server := grpc.NewServer()
	gop2pdme.RegisterP2PServiceServer(server, c)
	log.Printf("Server listening at %v", listen.Addr())

	go func() {
		if err := server.Serve(listen); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	for i := 0; i < int(peerCount); i++ {
		if i != int(c.id) {
			connect, err := grpc.Dial(fmt.Sprintf("localhosdt:%d", i+5000), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				log.Fatalf("Failed to connect %v", err)
			}

			c.peers[i] = gop2pdme.NewP2PServiceClient(connect)
			log.Printf("Client connected to peer %v", i)

			defer func(connect *grpc.ClientConn) {
				err := connect.Close()
				if err != nil {

				}
			}(connect)
		}
	}

	doneCritical := false
	for {
		if !doneCritical {
			if c.state != WANTED {
				c.state = WANTED
				c.Broadcast(REQUEST)
			} else if c.replies == int(peerCount-1) {
				c.lamport++
				c.state = HELD
				log.Println("inside of critical section")
				time.Sleep(5 * time.Second)
				log.Println("outside of critical section")
				c.state = RELEASED
				doneCritical = true
				c.Broadcast(DONE)
			}
		}
	}
}
