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
const (
    RELEASED = iota
    WANTED
    HELD
)

var (
    id int32 = -1
    state int = RELEASED
    lamport int64 = 0
    peersCount int32
    peers []peer
)

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
func (s *service) Recv(context context.Context, resp *gop2pdme.Post) (*gop2pdme.Empty, error) {
    log.Printf("Received %v message: %v", resp.Id, resp.Request)

    if resp.Lamport > lamport {
        lamport = resp.Lamport + 1
    } else {
        lamport++
    }

    peers[resp.Id].chanIn <-*resp
    return &gop2pdme.Empty{}, nil
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
func Send(peerId int32, request string) {
    lamport++
	_, err := peers[peerId].client.Recv(context.Background(), &gop2pdme.Post{Id: id, Request: request, Lamport: lamport}, grpc.WaitForReady(true))
    if err != nil {
		log.Fatalf("Failed to send %v a message: %v", peerId, err)
	}
}

func Broadcast(request string){
    for i, _ := range peers {
        if i != int(id) {
            Send(int32(i),request)
        }
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
func Critical() {
    reqQueue := make([]gop2pdme.Post,peersCount)
    doneCritical := false
    replies := 0

    go func(){
        for {
            for i, peer := range peers {
                if i != int(id) && len(peer.chanIn) > 0 {
                    req := <- peer.chanIn

                    if req.Request == "REQUEST" {
                        if state == HELD || (state == WANTED && int(id) < i) {
                            reqQueue = append(reqQueue,req)
                        } else {
                            Send(int32(i),"ALLOWED")
                        }
                        continue
                    }
                    replies++
                }
            }
        }
    }()

    go func(){
        for {
            if !doneCritical {
                if state != WANTED {
                    state = WANTED
                    Broadcast("REQUEST")
                } else if replies == int(peersCount)-1 {
                    state = HELD
                    log.Println("!!!INSIDE CRITICAL SECTION!!!")
                    state = RELEASED
                    doneCritical = true
                    for _, post := range reqQueue {
                        Send(post.Id,"ALLOWED")
                    }
                }
            }
        }
    }()
}

func main() {
	args := os.Args[1:] // args: <peer ID> <peer count>
	var pid, _ = strconv.ParseInt(args[0], 10, 32)
    var pcount, _ = strconv.ParseInt(args[1], 10, 32)
    id = int32(pid)
    peersCount = int32(pcount)

    Critical()
	StartClients()
	StartServer()
}
