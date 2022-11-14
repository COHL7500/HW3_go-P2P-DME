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

var pId int32
var pCount int32
var users []user

type user struct {
	id        int32
    chanDone  chan bool
    chanIn    chan gop2pdme.Post
    outStream gop2pdme.P2PService_RecvClient
    inStream  gop2pdme.P2PService_RecvServer
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

		users[resp.Id].chanIn <-*resp
	}
}

func StartServer() {
	// create listener
	lis, err := net.Listen("tcp",fmt.Sprintf("localhost:5%03d",pId))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// create grpc server
	ss := grpc.NewServer()
	gop2pdme.RegisterP2PServiceServer(ss, &service{})
	log.Printf("server listening at %v", lis.Addr())

	// launch server
	if err := ss.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// ---------------------------- //
// ---------- CLIENT ---------- //
// ---------------------------- //
func Send(outStream gop2pdme.P2PService_RecvClient) {
	usrIn := gop2pdme.Post{Id: pId, Message: "Hello!"}
	err := outStream.Send(&usrIn)
	if err != nil {
		return
	}
}

func StartClient() {
    for i := 0; i < int(pCount); i++ {
        if(i != int(pId)){
            users = append(users,user{int32(i),make(chan bool),make(chan gop2pdme.Post),nil,nil})
            go func(id int){
	            // dial server
	            conn, err := grpc.Dial(fmt.Sprintf("localhost:5%03d",id), grpc.WithTransportCredentials(insecure.NewCredentials()))
	            if err != nil {
		            log.Fatalf("Failed to connect %v", err)
	            }
	            client := gop2pdme.NewP2PServiceClient(conn)

	            // setup streams
	            outStream, outErr := client.Recv(context.Background(), grpc.WaitForReady(true))
	            if outErr != nil {
		            log.Fatalf("Failed to open message stream: %v", outErr)
	            }
                users[id].outStream = outStream
	            log.Printf("Client connected to peer %v", id)
                Send(outStream)

	            // closes server connection
	            select {
                    case <-users[id].chanDone:
                        log.Printf("Peer %v disconnected from client", id)
                        conn.Close()
                }
            }(i)
        } else {
            users = append(users,user{})
        }
    }
}

// -------------------------- //
// ---------- SETUP---------- //
// -------------------------- //
func main() {
	args := os.Args[1:] // args: <peer ID> <peer count>
	var pid, _ = strconv.ParseInt(args[0], 10, 32)
    var pcount, _ = strconv.ParseInt(args[1], 10, 32)
    pId = int32(pid)
    pCount = int32(pcount)

	StartClient()
	StartServer()
}
