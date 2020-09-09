package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	var addr string

	flag.StringVar(&addr, "addr", "", "Listen address")

	flag.Parse()

	if addr == "" {
		panic("addr cannot be empty")
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	helloworld.RegisterGreeterServer(server, &helloworldService{})
	grpc_health_v1.RegisterHealthServer(server, &healthService{})

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT)
	defer signal.Stop(interrupt)

	go func() {
		_ = server.Serve(listener)
	}()

	<-interrupt

	server.GracefulStop()
}

type healthService struct {
}

func (s *healthService) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	log.Println("Check", req.Service)

	return &grpc_health_v1.HealthCheckResponse{
		Status: grpc_health_v1.HealthCheckResponse_SERVING,
	}, nil
}

func (s *healthService) Watch(req *grpc_health_v1.HealthCheckRequest, w grpc_health_v1.Health_WatchServer) error {
	return nil
}

type helloworldService struct {
}

func (s *helloworldService) SayHello(ctx context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	log.Println("SayHello", req.Name)

	return &helloworld.HelloReply{Message: "Hello " + req.Name}, nil
}
