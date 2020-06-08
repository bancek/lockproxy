package lockproxy_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/onsi/ginkgo"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"google.golang.org/grpc/health/grpc_health_v1"
)

func NewLogger() *logrus.Logger {
	logger := logrus.New()
	logger.Out = ginkgo.GinkgoWriter
	logger.Level = logrus.DebugLevel

	formatter := &logrus.TextFormatter{
		TimestampFormat: time.RFC3339Nano,
	}
	logger.Formatter = formatter

	return logger
}

func NewLoggerEntry() *logrus.Entry {
	return NewLogger().WithFields(logrus.Fields{})
}

// Rand returns random 8 characters
func Rand() string {
	return uuid.New().String()[:8]
}

type healthServiceMock struct {
	mock.Mock
}

func (m *healthServiceMock) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	args := m.Called()
	return args.Get(0).(*grpc_health_v1.HealthCheckResponse), args.Error(1)
}

func (m *healthServiceMock) Watch(req *grpc_health_v1.HealthCheckRequest, w grpc_health_v1.Health_WatchServer) error {
	args := m.Called()
	return args.Error(0)
}
