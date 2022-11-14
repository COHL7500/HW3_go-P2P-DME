// ---------------------------- //
// ---------- IMPORT ---------- //
// ---------------------------- //
package main

import (
	"context"
	"log"
	"net"
	"os"
    "fmt"
	"strconv"

	"github.com/gop2pdme/proto"
	"google.golang.org/grpc"
)

// ----------------------------- //
// ---------- GLOBALS ---------- //
// ----------------------------- //
var id int32
var lamport int64
var peersCount int32
var peers []peer

type peer struct {
	id          int32
    chanDone    chan bool
    chanIn      chan gop2pdme.Post
    client      gop2pdme.P2PServiceClient
}

type service struct {
    gop2pdme.UnimplementedP2PServiceServer
}

// ---------------------------- //
// ---------- SERVER ---------- //
// ---------------------------- //
func (s *service) Recv(context context.Context, resp *gop2pdme.Post) (*gop2pdme.Post, error) {
    log.Printf("Received %v message: %v", resp.Id, resp.Message)
    peers[resp.Id].chanIn <-*resp
    return &gop2pdme.Post{}, nil
}

func StartServer() {
	// create listener
	lis, err := net.Listen("tcp",fmt.Sprintf("localhost:%d",id+5000))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// create grpc server
	server := grpc.NewServer()
	gop2pdme.RegisterP2PServiceServer(server, &service{})
	log.Printf("server listening at %v", lis.Addr())

	// launch server
	if err := server.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// ---------------------------- //
// ---------- CLIENT ---------- //
// ---------------------------- //
func Send(peerId int32, message string) {
	_, err := peers[peerId].client.Recv(context.Background(), &gop2pdme.Post{Id: id, Message: message}, grpc.WaitForReady(true))
    if err != nil {
		log.Fatalf("Failed to send %v a message: %v", peerId, err)
	}
}

func NewClient(peerId int32){
	// dial server
	conn, err := grpc.Dial(fmt.Sprintf("localhost:%d",peerId+5000), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to connect %v", err)
	}

    // setup client
	peers[peerId].client = gop2pdme.NewP2PServiceClient(conn)
	log.Printf("Client connected to peer %v", peerId)

	// closes connection
	select {
        case <-peers[peerId].chanDone:
            log.Printf("Peer %v disconnected from client", peerId)
            conn.Close()
	}
}

func StartClients() {
    for i := 0; i < int(peersCount); i++ {
        if i != int(id) {
            peers = append(peers,peer{int32(i),make(chan bool),make(chan gop2pdme.Post),nil})
            go NewClient(int32(i))
            continue
        }
        peers = append(peers,peer{})
    }
}

// -------------------------- //
// ---------- SETUP---------- //
// -------------------------- //
func main() {
	args := os.Args[1:] // args: <peer ID> <peer count>
	var pid, _ = strconv.ParseInt(args[0], 10, 32)
    var pcount, _ = strconv.ParseInt(args[1], 10, 32)
    id = int32(pid)
    peersCount = int32(pcount)

	StartClients()
	StartServer()
}
