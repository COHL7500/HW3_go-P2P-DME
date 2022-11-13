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
    clients map[string]gop2pdme.P2PServiceClient
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

func startServer(IPAddr string, IPPort string) {
    // create listener
    lis, err := net.Listen("tcp", IPAddr + ":" + IPPort)
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

func (s *server) startClient(name string, addr string) {
    // dial server
    conn, err := grpc.Dial(addr, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("Failed to connect %v", err)
    }

    s.clients[name] = gop2pdme.NewP2PServiceClient(conn)
    r, err := s.clients[name].Connect(context.Background(), &gop2pdme.Post{Id: pId})
    
}

func main() {
    args := os.Args[1:]

}
