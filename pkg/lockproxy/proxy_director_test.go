package lockproxy_test

import (
	"net"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"

	. "github.com/bancek/lockproxy/pkg/lockproxy"
)

var _ = Describe("ProxyDirector", func() {
	createHealthServer := func() (string, *healthServiceMock, func()) {
		healthListener, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())

		healthAddr := healthListener.Addr().String()

		healthService := new(healthServiceMock)
		healthService.On("Check").Return(&grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_SERVING,
		}, nil)
		healthServer := NewHealthServer(healthService)
		go func() {
			_ = healthServer.Serve(healthListener)
		}()

		return healthAddr, healthService, func() {
			healthServer.Stop()
			healthListener.Close()
		}
	}

	createDirector := func(
		upstreamAddrProvider UpstreamAddrProvider,
		healthAddr string,
		grpcMaxCallRecvMsgSize int,
		grpcMaxCallSendMsgSize int,
		abortTimeout time.Duration,
	) (grpc_health_v1.HealthClient, func()) {
		grpcDialTransportSecurity := grpc.WithInsecure()

		proxyListener, err := net.Listen("tcp", "127.0.0.1:0")
		Expect(err).NotTo(HaveOccurred())

		proxyAddr := proxyListener.Addr().String()

		proxyDirector := NewProxyDirector(
			TestCtx,
			upstreamAddrProvider,
			healthAddr,
			grpcDialTransportSecurity,
			grpcMaxCallRecvMsgSize,
			grpcMaxCallSendMsgSize,
			abortTimeout,
			Logger,
		)
		proxyServer := NewProxyServer(proxyDirector)
		go func() {
			_ = proxyServer.Serve(proxyListener)
		}()

		conn, err := grpc.DialContext(TestCtx, proxyAddr, grpc.WithInsecure())
		Expect(err).NotTo(HaveOccurred())

		healthClient := grpc_health_v1.NewHealthClient(conn)

		return healthClient, func() {
			conn.Close()
			proxyServer.Stop()
			proxyListener.Close()
		}
	}

	It("should proxy health to upstream if leader", func() {
		healthAddr1, healthServiceMock1, healthStop1 := createHealthServer()
		defer healthStop1()
		healthAddr2, healthServiceMock2, healthStop2 := createHealthServer()
		defer healthStop2()
		upstreamAddrProvider := func() (addr string, isLeader bool) {
			return healthAddr1, true
		}
		healthClient, stop := createDirector(
			upstreamAddrProvider,
			healthAddr2,
			4*1024*1024,
			4*1024*1024,
			10*time.Second,
		)
		defer stop()

		resp, err := healthClient.Check(TestCtx, &grpc_health_v1.HealthCheckRequest{})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.Status).To(Equal(grpc_health_v1.HealthCheckResponse_SERVING))

		healthServiceMock1.AssertCalled(GinkgoT(), "Check")
		healthServiceMock2.AssertNumberOfCalls(GinkgoT(), "Check", 0)
	})

	It("should proxy health to local server if not leader", func() {
		healthAddr1, healthServiceMock1, healthStop1 := createHealthServer()
		defer healthStop1()
		healthAddr2, healthServiceMock2, healthStop2 := createHealthServer()
		defer healthStop2()
		upstreamAddrProvider := func() (addr string, isLeader bool) {
			return healthAddr1, false
		}
		healthClient, stop := createDirector(
			upstreamAddrProvider,
			healthAddr2,
			4*1024*1024,
			4*1024*1024,
			10*time.Second,
		)
		defer stop()

		resp, err := healthClient.Check(TestCtx, &grpc_health_v1.HealthCheckRequest{})
		Expect(err).NotTo(HaveOccurred())
		Expect(resp.Status).To(Equal(grpc_health_v1.HealthCheckResponse_SERVING))

		healthServiceMock1.AssertNumberOfCalls(GinkgoT(), "Check", 0)
		healthServiceMock2.AssertCalled(GinkgoT(), "Check")
	})

	It("should fail if response is larger than grpcMaxCallRecvMsgSize", func() {
		healthAddr, healthServiceMock, healthStop := createHealthServer()
		defer healthStop()
		upstreamAddrProvider := func() (addr string, isLeader bool) {
			return healthAddr, true
		}
		healthClient, stop := createDirector(
			upstreamAddrProvider,
			healthAddr,
			1,
			4*1024*1024,
			10*time.Second,
		)
		defer stop()

		_, err := healthClient.Check(TestCtx, &grpc_health_v1.HealthCheckRequest{})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("received message larger than max"))

		healthServiceMock.AssertNumberOfCalls(GinkgoT(), "Check", 1)
	})

	It("should fail if request is larger than grpcMaxCallSendMsgSize", func() {
		healthAddr, healthServiceMock, healthStop := createHealthServer()
		defer healthStop()
		upstreamAddrProvider := func() (addr string, isLeader bool) {
			return healthAddr, true
		}
		healthClient, stop := createDirector(
			upstreamAddrProvider,
			healthAddr,
			4*1024*1024,
			1,
			10*time.Second,
		)
		defer stop()

		_, err := healthClient.Check(TestCtx, &grpc_health_v1.HealthCheckRequest{
			Service: strings.Repeat("a", 1*1024*1024),
		})
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("trying to send message larger than max"))

		healthServiceMock.AssertNumberOfCalls(GinkgoT(), "Check", 0)
	})
})
