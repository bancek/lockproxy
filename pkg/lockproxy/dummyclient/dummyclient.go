package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/helloworld/helloworld"
)

func main() {
	var addr string

	flag.StringVar(&addr, "addr", "", "Listen address")

	flag.Parse()

	if addr == "" {
		panic("addr cannot be empty")
	}

	conn, err := grpc.DialContext(context.Background(), addr, grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	client := helloworld.NewGreeterClient(conn)

	resp, err := client.SayHello(context.Background(), &helloworld.HelloRequest{
		Name: strings.Join(flag.Args(), " "),
	})
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Message)
}
