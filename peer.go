// ---------------------------- //
// ---------- IMPORT ---------- //
// ---------------------------- //
package main

import (
	"context"
	"google.golang.org/grpc/credentials/insecure"
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
var peers int32
var lamport int64
var users []user

type user struct {
	id          int32
    chanDone    chan bool
    chanIn      chan gop2pdme.Post
    outStream   gop2pdme.P2PService_RecvClient
}

type service struct {
    gop2pdme.UnimplementedP2PServiceServer
}

// ---------------------------- //
// ---------- SERVER ---------- //
// ---------------------------- //
func (s *service) Recv(inStream gop2pdme.P2PService_RecvServer) error {
	for {
		resp, err := inStream.Recv()

		if err != nil {
			log.Fatalf("Failed to receive %v", err)
			return nil
		}

        if resp.Message == "disconnect" {
            users[resp.Id].chanDone <- true
            log.Printf("Peer %v disconnected from the server...", resp.Id)
            return nil
        }

        log.Printf("Received %v message: %v", resp.Id, resp.Message)
		users[resp.Id].chanIn <-*resp
	}
}

func StartServer() {
	// create listener
	lis, err := net.Listen("tcp",fmt.Sprintf("localhost:5%03d",id))
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
	if err := users[peerId].outStream.Send(&gop2pdme.Post{Id: id, Message: message}); err != nil {
		log.Fatalf("Failed to send %v a message: %v", peerId, err)
	}
}

func NewClient(peerId int32){
	// dial server
	conn, err := grpc.Dial(fmt.Sprintf("localhost:5%03d",peerId), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect %v", err)
	}
	client := gop2pdme.NewP2PServiceClient(conn)

	// setup streams
	outStream, outErr := client.Recv(context.Background(), grpc.WaitForReady(true))
	if outErr != nil {
		log.Fatalf("Failed to open message stream: %v", outErr)
	}
    users[peerId].outStream = outStream
	log.Printf("Client connected to peer %v", peerId)
    Send(peerId,"Hello!")

	// closes server connection
	select {
        case <-users[peerId].chanDone:
            log.Printf("Peer %v disconnected from client", peerId)
            conn.Close()
	}
}

func StartClients() {
    for i := 0; i < int(peers); i++ {
        if i != int(id) {
            users = append(users,user{int32(i),make(chan bool),make(chan gop2pdme.Post),nil})
            go NewClient(int32(i))
            continue
        }
        users = append(users,user{})
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
    peers = int32(pcount)

	StartClients()
	StartServer()
}
