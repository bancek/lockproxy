package lockproxy

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func NewHealthServer(healthService grpc_health_v1.HealthServer) *grpc.Server {
	server := grpc.NewServer()

	grpc_health_v1.RegisterHealthServer(server, healthService)

	return server
}
