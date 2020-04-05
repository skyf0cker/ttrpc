package main

import (
	"context"
	"fmt"
	"github.com/containerd/ttrpc"
	"github.com/containerd/ttrpc/example/stream/pb"
	"log"
	"net"
)

type server struct {

}

func (s *server)AttachContainer(containerServer pb.Islands_AttachContainerServer) error {
	for {
		data, _ := containerServer.Recv()
		fmt.Println("server", string(data.Content))
		_ = containerServer.Send(&pb.AttachStreamOut{
			Content: data.Content,
		})
	}
	return nil
}

func (s *server)ListContainer(ctx context.Context, req *pb.ContainerInfo) (*pb.Result, error) {
	return &pb.Result{
		Status: 520,
	}, nil
}

func main() {
	s, err := net.Listen("tcp", "127.0.0.1:8000")
	if err != nil {
		log.Fatalln(err)
	}

	rpcServer, err := ttrpc.NewServer()
	if err != nil {
		log.Fatalln(err)
	}

	pb.RegisterIslandsService(rpcServer, &server{})
	if err := rpcServer.Serve(context.Background(), s); err != nil {
		log.Fatalln(err)
	}
}
