package main

import (
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

// The main execution file.

func main() {

	// receives the arguments given when running the program:
	peerId, _ := strconv.ParseInt(os.Args[1], 10, 32)    // Id of the peer, for example "0", "1" and "2".
	peerCount, _ := strconv.ParseInt(os.Args[2], 10, 32) // Amount of peers, for example "3".

	// Initiate a client based on the "client" struct.
	client := &client{
		id:       int32(peerId),                                           // Client's ID.
		state:    RELEASED,                                                // Current state of critical section. Default is RELEASED.
		lamport:  0,                                                       // Lamport time. Always starts with 0.
		peers:    make(map[int]gop2pdme.P2PServiceClient, int(peerCount)), // Makes new map containing all peers.
		reqQueue: make([]gop2pdme.Post, 0),                                // Makes new slice of length 0.
		replies:  0,                                                       // Count of replies received.
	}

	// Assign listen. If error occurs, stop process.
	listen, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", int(client.id)+5000))

	if err != nil {
		log.Fatalf("Failed to listen %v", listen.Addr())
	}

	// make and register new server.
	server := grpc.NewServer()
	gop2pdme.RegisterP2PServiceServer(server, client)
	log.Printf("Server listening at %v", listen.Addr())

	// concurrently check whether an error occurs.
	// server.Serve reads and accepts gRPC requests. It only returns if an error occurs.
	go func() {
		if err := server.Serve(listen); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Loop through all peers
	for i := 0; i < int(peerCount); i++ {

		// If i doesn't correspond to the current client's ID, connect to it.
		if i != int(client.id) {
			connect, err := grpc.Dial(fmt.Sprintf("localhosdt:%d", i+5000), grpc.WithTransportCredentials(insecure.NewCredentials()))

			if err != nil {
				log.Fatalf("Failed to connect %v", err)
			}

			// Add the peer to the list of peers client has.
			client.peers[i] = gop2pdme.NewP2PServiceClient(connect)

			log.Printf("Client connected to peer %v", i)

			// closes whenever the surrounding function returns.
			defer func(connect *grpc.ClientConn) {
				err := connect.Close()
				if err != nil {
					log.Fatalf("Failed to close connection: %v", err)
				}
			}(connect)
		}
	}

	// Boolean for checking whether peer is done checking the critical section or not.
	doneCritical := false

	// Loop for handling Ricart & Agrawala Mutual Exclusion
	for {

		// if peer hasn't entered critical section yet, inform other peers it is wanted.
		if !doneCritical {
			if client.state != WANTED {
				client.state = WANTED
				client.Broadcast(REQUEST)

				// If the peer received replies from all peers, access critical section.
			} else if client.replies == int(peerCount-1) {
				client.lamport++
				client.state = HELD
				log.Println("inside of critical section")
				time.Sleep(5 * time.Second)
				log.Println("outside of critical section")
				client.state = RELEASED
				doneCritical = true
				client.Broadcast(DONE)
			}
		}
	}
}
