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
    chanOut     chan gop2pdme.Post
}

type service struct {
    gop2pdme.UnimplementedP2PServiceServer
}

// ---------------------------- //
// ---------- SERVER ---------- //
// ---------------------------- //
func (s *service) Recv(context context.Context, resp *gop2pdme.Post) (*gop2pdme.Empty, error) {
    if resp.Lamport > lamport {
        lamport = resp.Lamport + 1
    } else {
        lamport++
    }

    log.Printf("Recieved message %v at %v by %d", resp.Request, lamport, resp.Id)
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
    peers[peerId].chanOut <- gop2pdme.Post{Id: id, Request: request, Lamport: lamport}
}

func Broadcast(request string){
    for i, _ := range peers {
        if int32(i) != id {
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
	client := gop2pdme.NewP2PServiceClient(conn)
	log.Printf("Client connected to peer %v", peerId)

	// closes connection
    for {
	    select {
            case p := <-peers[peerId].chanOut:
		        lamport++
                log.Printf("Sendt message %v at %v", p.Request, lamport)
                _, err := client.Recv(context.Background(), &p, grpc.WaitForReady(true))
                if err != nil {
		            log.Fatalf("Failed to send %v a message: %v", peerId, err)
	            }
            case <-peers[peerId].chanDone:
                log.Printf("Peer %v disconnected from client", peerId)
                conn.Close()
                return
	    }
    }
}

func StartClients() {
    for i := 0; i < int(peersCount); i++ {
        if i != int(id) {
            peers = append(peers,peer{int32(i),make(chan bool,1),make(chan gop2pdme.Post,1),make(chan gop2pdme.Post,1)})
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
    reqQueue := make([]gop2pdme.Post,0)
    doneCritical := false
    replies := 0

    go func(){
        for {
            for i, peer := range peers {
                if i != int(id) && len(peer.chanIn) > 0 {
                    req := <-peer.chanIn

                    if req.Request == "REQUEST" {
                        if state == HELD || (state == WANTED && int(id) < i) {
                            reqQueue = append(reqQueue,req)
                            continue
                        }
                        Send(int32(i),"ALLOWED")
                        continue
                    }
                    replies++
                }
            }
        }
    }()

    for {
        if !doneCritical {
            if state != WANTED {
                state = WANTED
                Broadcast("REQUEST")
            } else if replies == int(peersCount)-1 {
                lamport++
                state = HELD
                log.Printf("!!!INSIDE CRITICAL SECTION!!! %v", lamport)
                state = RELEASED
                doneCritical = true
                for _, post := range reqQueue {
                    Send(post.Id,"ALLOWED")
                }
            }
        }
    }
}

func main() {
	args := os.Args[1:] // args: <peer ID> <peer count>
	var pid, _ = strconv.ParseInt(args[0], 10, 32)
    var pcount, _ = strconv.ParseInt(args[1], 10, 32)
    id = int32(pid)
    peersCount = int32(pcount)

	StartClients()
	go StartServer()
    Critical()
}
