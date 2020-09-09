package lockproxy

import (
	"context"

	"google.golang.org/grpc/health/grpc_health_v1"
)

type HealthService struct {
	isLeader func() bool
}

func NewHealthService(isLeader func() bool) *HealthService {
	return &HealthService{
		isLeader: isLeader,
	}
}

func (s *HealthService) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	status := grpc_health_v1.HealthCheckResponse_SERVING

	if req.Service == "lockproxyleader" && !s.isLeader() {
		status = grpc_health_v1.HealthCheckResponse_NOT_SERVING
	}

	return &grpc_health_v1.HealthCheckResponse{
		Status: status,
	}, nil
}

func (s *HealthService) Watch(req *grpc_health_v1.HealthCheckRequest, w grpc_health_v1.Health_WatchServer) error {
	return nil
}
