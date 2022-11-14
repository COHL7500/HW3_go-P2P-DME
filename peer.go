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
	"strconv"

	"github.com/gop2pdme/proto"
	"google.golang.org/grpc"
)

var pId int32

// ---------------------------- //
// ---------- SERVER ---------- //
// ---------------------------- //
type user struct {
	id       int32
	chanCom  chan gop2pdme.Post
	chanDone chan bool
}

type server struct {
	users []*user
	gop2pdme.UnimplementedP2PServiceServer
}

func (s *server) Connect(in *gop2pdme.Post, srv gop2pdme.P2PService_ConnectServer) error {
	userData := user{in.Id, make(chan gop2pdme.Post), make(chan bool)}
	s.users = append(s.users, &userData)
	err := srv.Send(&gop2pdme.Post{Id: userData.id})
	if err != nil {
		return err
	}
	log.Printf("Peer %v connected to the server...", in.Id)

	for {
		select {
		// Variable declaration copies a lock value to 'm': type 'gop2pdme.Post' contains 'protoimpl.MessageState' contains 'sync.Mutex' which is 'sync.Locker'
		case m := <-userData.chanCom:
			err := srv.Send(&m)
			if err != nil {
				return err
			}
		case <-userData.chanDone:
			s.users[userData.id] = nil
			return nil
		}
	}
}

func (s *server) Disconnect(context context.Context, in *gop2pdme.Post) (out *gop2pdme.Empty, err error) {
	usr := s.users[in.Id]
	usr.chanDone <- true
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

func (s *server) start(id int32, addr string) {
	if addr == "nil" {
		log.Printf("Failed to establish server: addr was empty")
		return
	}

	// create listener
	pId = id
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// create grpc server
	ss := grpc.NewServer()
	gop2pdme.RegisterP2PServiceServer(ss, &server{})
	log.Printf("server listening at %v", lis.Addr())

	// launch server
	go func() {
		if err := ss.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
}

// ---------------------------- //
// ---------- CLIENT ---------- //
// ---------------------------- //
type client struct {
	clients map[int32]gop2pdme.P2PServiceClient
}

func (c *client) inPost(done chan bool, inStream gop2pdme.P2PService_ConnectClient) {
	for {
		resp, err := inStream.Recv()

		if err == io.EOF {
			done <- true
			return
		}

		if err != nil {
			log.Fatalf("Failed to receive %v", err)
		}

		log.Printf("received the message: %v", resp.Message)
	}
}

func (c *client) outPost(outStream gop2pdme.P2PService_MessagesClient) {
	usrIn := gop2pdme.Post{Id: pId}
	err := outStream.Send(&usrIn)
	if err != nil {
		return
	}
}

func (c *client) start(id int32, addr string) {
	if addr == "nil" {
		log.Printf("Failed to establish client side: addr was empty")
		return
	}

	// dial server
	c.clients = make(map[int32]gop2pdme.P2PServiceClient)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect %v", err)
	}
	c.clients[id] = gop2pdme.NewP2PServiceClient(conn)

	// setup streams
	inStream, inErr := c.clients[id].Connect(context.Background(), &gop2pdme.Post{Id: pId}, grpc.WaitForReady(true))
	if inErr != nil {
		log.Fatalf("Failed to open connection stream: %v", inErr)
	}
	outStream, outErr := c.clients[id].Messages(context.Background())
	if outErr != nil {
		log.Fatalf("Failed to open message stream: %v", outErr)
	}
	log.Printf("Client connected to peer %v (%v)", id, addr)

	// running goroutines streams
	done := make(chan bool)
	go c.inPost(done, inStream)
	go c.outPost(outStream)
	<-done

	// closes server connection
	log.Printf("Closing peer connection %v", addr)
	err = conn.Close()
	if err != nil {
		return
	}
	log.Printf("Peer connection ended %v", addr)
}

// -------------------------- //
// ---------- SETUP---------- //
// -------------------------- //
func main() {
	// args: peer ID, peer server IP/PORT, neighbor ID, neighbor IP/PORT
	args := os.Args[1:]

	if len(args) < 4 {
		log.Println("Arguments required: <peer ID> <peer IP:PORT> <neighbor ID> <neighbor IP:PORT>")
		os.Exit(1)
	}

	var i, _ = strconv.ParseInt(args[0], 10, 32)
	var j, _ = strconv.ParseInt(args[2], 10, 32)

	fin := make(chan bool)
	go (&server{}).start(int32(i), args[1])
	go (&client{}).start(int32(j), args[3])
	<-fin
}
