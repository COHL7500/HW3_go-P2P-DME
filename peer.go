package main

import (
    "context"
    "net"
    "log"
    "io"
    "fmt"
    "strings"

    "github.com/gop2pdme/proto"
    "google.golang.org/grpc"
)

pId int32;

type user struct {
    id int32
    chanCom chan goChittyChat.Post
    chanDone chan bool
}

type server struct {
    gop2pdme.UnimplementedP2PServiceServer
    users []*user
    clients map[int32]gop2pdme.P2PServiceClient
}

func (s *server) Connect(in *gop2pdme.Post, srv gop2pdme.P2PService_ConnectServer) error {
    userData := user{in.Id, make(chan gop2pdme.Post), make(chan bool)}
    s.users = append(s.users, &userData)
    srv.Send(&gop2pdme.Post{Id: userData.id, Messages: nil})

    for {
        select {
            case m := userData.chanCom:
                srv.Send(&m)
            case <-userData.chanDone:
                s.users[userData.id] = nil
                return nil
        }
    }
}

func (s *server) Disconnect(context context.Context, in *gop2pdme.Post) (out *gop2pdme.Empty, err error) {
    usr := s.users[in.Id]
    usr.chanDone <- true
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
    }
}

func inPost(done chan bool, inStream gop2pdme.P2PService_ConnectClient){
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

func outPost(done chan bool, outStream gop2pdme.P2PService_MessageClient){
    usrIn := gop2pdme.Post{Id: pId, Message: nil}
    outStream.send(&usrIn)
}

func startServer(addr string) {
    // create listener
    lis, err := net.Listen("tcp", addr)
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    // create grpc server
    s := grpc.NewServer()
    gop2pdme.RegisterChatServiceServer(s, &server{})
    log.Printf("server listening at %v", lis.Addr())

    // launch server
    if err := s.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}

func (s *server) startClient(id int32, addr string) {
    // dial server
    conn, err := grpc.Dial(addr, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("Failed to connect %v", err)
    }
    s.clients[name] = gop2pdme.NewP2PServiceClient(conn)

    // setup streams
    inStream, inErr := s.clients[name].Connect(context.Background(), &gop2pdme.Post{Id: pId})
    if inErr != nil {
        log.Fatalf("Failed to open connection stream: %v", inErr)
    }
    outStream, outErr := s.clients[name].Messages(context.Background())
    if outErr != nil {
        log.Fatalf("Failed to open message stream: %v", outErr)
    }
    log.Println("Client connected to peer %v", addr)

    // running goroutines streams
    done := make(chan bool)
    go inPost(done, inStream)
    go outPost(done, outStream)
    <-done

    // closes server connection
    log.Printf("Closing peer connection %v", addr)
    conn.Close()
    log.Printf("Peer connection ended %v", addr)
}

func main() {
    args := os.Args[1:]
    startServer(args[0])
    startClient()
}
