package main

import (
	"context"
	"fmt"
	"github.com/containerd/ttrpc"
	"github.com/containerd/ttrpc/example/stream/pb"
	"log"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:8000")
	if err != nil {
		log.Fatalln(err)
	}
	client := ttrpc.NewClient(conn)
	islandsClient := pb.NewIslandsClient(client)
	stream := islandsClient.AttachContainer(context.Background())
	_ = stream.Send(&pb.AttachStreamIn{
		Id:      "12",
		Content: []byte("hello ttrpc stream"),
	})

	msg, err := stream.Recv()
	fmt.Println(string(msg.Content))
}
