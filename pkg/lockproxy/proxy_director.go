package lockproxy

import (
	"context"
	"strings"
	"time"

	"github.com/mwitkow/grpc-proxy/proxy"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UpstreamAddrProvider func() (addr string, isLeader bool)

// ProxyDirector is gRPC connection handler.
// It is called for every gRPC method call. It gets the current upstream
// address from upstreamAddrProvider and fails with Unavailable if
// the address is not set. It accepts a global Context after which
// all the requests are cancelled. If the current method is
// /grpc.health.v1.Health/* and the server is not leader we proxy
// the request to the our internal health server.
type ProxyDirector struct {
	ctx                       context.Context
	upstreamAddrProvider      UpstreamAddrProvider
	healthAddr                string
	grpcDialTransportSecurity grpc.DialOption
	grpcMaxCallRecvMsgSize    int
	grpcMaxCallSendMsgSize    int
	abortTimeout              time.Duration
	logger                    *logrus.Entry
}

func NewProxyDirector(
	ctx context.Context,
	upstreamAddrProvider UpstreamAddrProvider,
	healthAddr string,
	grpcDialTransportSecurity grpc.DialOption,
	grpcMaxCallRecvMsgSize int,
	grpcMaxCallSendMsgSize int,
	abortTimeout time.Duration,
	logger *logrus.Entry,
) *ProxyDirector {
	return &ProxyDirector{
		ctx:                       ctx,
		upstreamAddrProvider:      upstreamAddrProvider,
		healthAddr:                healthAddr,
		grpcDialTransportSecurity: grpcDialTransportSecurity,
		grpcMaxCallRecvMsgSize:    grpcMaxCallRecvMsgSize,
		grpcMaxCallSendMsgSize:    grpcMaxCallSendMsgSize,
		abortTimeout:              abortTimeout,
		logger:                    logger,
	}
}

func (d *ProxyDirector) isHealth(fullMethodName string) bool {
	return strings.HasPrefix(fullMethodName, "/grpc.health.v1.Health/")
}

func (d *ProxyDirector) Director(ctx context.Context, fullMethodName string) (context.Context, *grpc.ClientConn, error) {
	outCtx, cancel := context.WithCancel(ctx)
	go func() {
		select {
		case <-d.ctx.Done():
			d.logger.WithField("abortTimeout", d.abortTimeout).Info("ProxyDirector aborting request")
			time.AfterFunc(d.abortTimeout, cancel)
		case <-ctx.Done():
		}
	}()

	isHealth := d.isHealth(fullMethodName)

	addr, isLeader := d.upstreamAddrProvider()

	if isHealth && !isLeader {
		addr = d.healthAddr
	}

	if addr == "" {
		d.logger.Warn("ProxyDirector no address")
		return nil, nil, status.Error(codes.Unavailable, "no master")
	}

	d.logger.WithField("addr", addr).Debug("ProxyDirector proxy request")

	clientConn, err := grpc.DialContext(
		outCtx,
		addr,
		grpc.WithDefaultCallOptions(
			grpc.CustomCodecCallOption{Codec: proxy.Codec()},
			grpc.MaxCallRecvMsgSize(d.grpcMaxCallRecvMsgSize),
			grpc.MaxCallSendMsgSize(d.grpcMaxCallSendMsgSize),
		),
		d.grpcDialTransportSecurity,
	)
	if err != nil {
		d.logger.WithFields(logrus.Fields{
			"addr":          addr,
			logrus.ErrorKey: err,
		}).Warn("ProxyDirector proxy dial error")
		return outCtx, nil, err
	}

	return outCtx, clientConn, nil
}
