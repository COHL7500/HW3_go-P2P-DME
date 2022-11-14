// ---------------------------- //
// ---------- IMPORT ---------- //
// ---------------------------- //
package main

import (
	"context"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"net"
	"os"
    "fmt"
	"strconv"

	"github.com/gop2pdme/proto"
	"google.golang.org/grpc"
)

var pId int32
var pCount int32
var users []user

// ---------------------------- //
// ---------- SERVER ---------- //
// ---------------------------- //
type user struct {
	id        int32
    chanDone  chan bool
    chanIn    chan gop2pdme.Post
    outStream gop2pdme.P2PService_MessagesClient
}

type server struct {
	gop2pdme.UnimplementedP2PServiceServer
}

func (s *server) Connect(in *gop2pdme.Post, srv gop2pdme.P2PService_ConnectServer) error {
	log.Printf("Peer %v connected to the server...", in.Id)
	for {
		select {
		    // Variable declaration copies a lock value to 'm': type 'gop2pdme.Post' contains 'protoimpl.MessageState' contains 'sync.Mutex' which is 'sync.Locker'
		    case m := <-users[in.Id].chanIn:
			    err := srv.Send(&m)
			    if err != nil {
				    return err
			    }
		    case <-users[in.Id].chanDone:
			    return nil
		}
	}
}

func (s *server) Disconnect(_ context.Context, in *gop2pdme.Post) (out *gop2pdme.Empty, err error) {
	users[in.Id].done = true
	log.Printf("Peer %v disconnected from the server...", in.Id)
	return &gop2pdme.Empty{}, nil
}

func (s *server) Messages(srv gop2pdme.P2PService_MessagesServer) error {
	for {
		resp, err := srv.Recv()

		if err == io.EOF {
			return nil
		}

		if err != nil {
			log.Fatalf("Failed to receive %v", err)
			return nil
		}

		log.Println(resp)
	}
}

func startServer(id int32) {
	// create listener
	lis, err := net.Listen("tcp", + fmt.Sprintf("localhost:5%03d",pId))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// create grpc server
	ss := grpc.NewServer()
	gop2pdme.RegisterP2PServiceServer(ss, &server{})
	log.Printf("server listening at %v", lis.Addr())

	// launch server
	if err := ss.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// ---------------------------- //
// ---------- CLIENT ---------- //
// ---------------------------- //
func send(id int32) {
	usrIn := gop2pdme.Post{Id: pId}
	err := users[id].outStream.Send(&usrIn)
	if err != nil {
		return
	}
}

func startClient() {
    for i := 0; i < pCount; i++ {
        if(i != pId){
            users = append(users,user{i,make(chan bool),make(chan gop2pdme.Post),nil})
            go func(id int32){
	            // dial server
	            conn, err := grpc.Dial(fmt.Sprintf("localhost:5%03d",id), grpc.WithTransportCredentials(insecure.NewCredentials()))
	            if err != nil {
		            log.Fatalf("Failed to connect %v", err)
	            }
	            client = gop2pdme.NewP2PServiceClient(conn)

	            // setup streams
	            outStream, outErr := client.Messages(context.Background(), grpc.WaitForReady(true))
	            if outErr != nil {
		            log.Fatalf("Failed to open message stream: %v", outErr)
	            }
                users[id].outStream = outStream
	            log.Printf("Client connected to peer %v (%v)", id, addr)

	            // closes server connection
	            select {
                    case <-users[id].chanDone:
                        log.Printf("Closing peer connection %v", id)
                        conn.Close()
                        log.Printf("Peer connection ended %v", id)
                }
            }(i)
        }
    }
}

// -------------------------- //
// ---------- SETUP---------- //
// -------------------------- //
func main() {
	args := os.Args[1:] // args: <peer ID> <peer count>
	var pcount, _ = strconv.ParseInt(args[0], 10, 32)
	var pid, _ = strconv.ParseInt(args[2], 10, 32)
    pCount = int32(pcount)
    pId = int32(pid)

	startClient()
	startServer()
}
