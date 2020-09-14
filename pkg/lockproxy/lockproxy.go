package lockproxy

import (
	"context"
	"net"
	"net/http"

	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
	"google.golang.org/grpc"

	"github.com/bancek/lockproxy/pkg/lockproxy/config"
)

type LockProxy struct {
	config  *config.Config
	adapter Adapter
	logger  *logrus.Entry

	ctx             context.Context
	cancel          func()
	errGroup        *errgroup.Group
	debugListener   net.Listener
	healthListener  net.Listener
	proxyListener   net.Listener
	pinger          *Pinger
	localAddrStore  LocalAddrStore
	remoteAddrStore RemoteAddrStore
	proxyDirector   *ProxyDirector
	commander       *Commander
	locker          Locker
	debugServer     *http.Server
	healthService   *HealthService
	healthServer    *grpc.Server
	proxyServer     *grpc.Server
}

func NewLockProxy(
	config *config.Config,
	adapter Adapter,
	logger *logrus.Entry,
) *LockProxy {
	return &LockProxy{
		config:  config,
		adapter: adapter,
		logger:  logger,
	}
}

func (p *LockProxy) Init(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	errGroup, ctx := errgroup.WithContext(ctx)
	p.ctx = ctx
	p.cancel = cancel
	p.errGroup = errGroup

	grpcDialTransportSecurity, err := GrpcDialTransportSecurity(
		p.config.UpstreamClientCert,
		p.config.UpstreamClientKey,
		p.config.UpstreamClientCA,
	)
	if err != nil {
		return xerrors.Errorf("failed to setup grpc dial transport security: %w", err)
	}

	p.debugListener, err = net.Listen("tcp", p.config.DebugListenAddr)
	if err != nil {
		return xerrors.Errorf("debug listener listen failed: %s: %w", p.config.DebugListenAddr, err)
	}

	p.healthListener, err = net.Listen("tcp", p.config.HealthListenAddr)
	if err != nil {
		return xerrors.Errorf("health listener listen failed: %s: %w", p.config.HealthListenAddr, err)
	}

	p.proxyListener, err = net.Listen("tcp", p.config.ProxyListenAddr)
	if err != nil {
		return xerrors.Errorf("proxy listener listen failed: %s: %w", p.config.ProxyListenAddr, err)
	}

	err = p.adapter.Init(ctx)
	if err != nil {
		return xerrors.Errorf("failed to init adapter: %w", err)
	}

	p.localAddrStore = NewMemoryAddrStore(p.logger)

	remoteAddrStore, err := p.adapter.GetRemoteAddrStore(p.localAddrStore)
	if err != nil {
		return xerrors.Errorf("failed to get remote addr store: %w", err)
	}
	p.remoteAddrStore = remoteAddrStore

	p.pinger = NewPinger(
		p.ping,
		p.config.PingRetryInitialInterval,
		p.config.PingRetryMaxElapsedTime,
		p.config.PingDelay,
		p.config.PingInitialDelay,
		p.logger,
	)

	locker, err := p.adapter.GetLocker(p.onLocked)
	if err != nil {
		return xerrors.Errorf("failed to get locker: %w", err)
	}
	p.locker = locker

	p.proxyDirector = NewProxyDirector(
		ctx,
		p.upstreamAddrProvider,
		p.config.HealthListenAddr,
		grpcDialTransportSecurity,
		p.config.ProxyGrpcMaxCallRecvMsgSize,
		p.config.ProxyGrpcMaxCallSendMsgSize,
		p.config.ProxyRequestAbortTimeout,
		p.config.ProxyHealthFollowerInternal,
		p.logger,
	)

	p.commander = NewCommander(p.config.Cmd, p.config.CmdShutdownTimeout, p.logger)

	p.debugServer = NewDebugServer()

	p.healthService = NewHealthService(p.isLeader)
	p.healthServer = NewHealthServer(p.healthService)

	p.proxyServer = NewProxyServer(p.proxyDirector)

	return nil
}

func (p *LockProxy) upstreamAddrProvider(ctx context.Context) (addr string, isLeader bool) {
	addr = p.remoteAddrStore.Addr(ctx)
	isLeader = addr == p.config.UpstreamAddr
	return addr, isLeader
}

func (p *LockProxy) isLeader(ctx context.Context) bool {
	addr := p.remoteAddrStore.Addr(ctx)
	return addr == p.config.UpstreamAddr
}

func (p *LockProxy) ping(ctx context.Context) error {
	_, err := p.remoteAddrStore.Refresh(ctx)
	return err
}

func (p *LockProxy) onLocked(ctx context.Context) error {
	err := p.remoteAddrStore.SetAddr(ctx, p.config.UpstreamAddr)
	if err != nil {
		return xerrors.Errorf("failed to set addr: %w", err)
	}

	return p.commander.Start(ctx)
}

func (p *LockProxy) Spawn(name string, f func(context.Context) error) {
	logger := p.logger.WithField("name", name)

	p.errGroup.Go(func() error {
		logger.Debug("Spawn start")

		err := f(p.ctx)
		if err != nil {
			logger.WithError(err).Debug("Spawn done")

			p.cancel()

			return err
		}

		logger.Debug("Spawn done")

		return nil
	})
}

func (p *LockProxy) Start() error {
	p.Spawn("debugHTTPServer", func(ctx context.Context) error {
		p.logger.WithField("listenAddr", p.config.DebugListenAddr).Info("Starting debug HTTP server")

		err := p.debugServer.Serve(p.debugListener)
		if ctx.Err() == nil {
			return err
		}
		return nil
	})

	addrStoreStarted := make(chan struct{}, 1)

	p.logger.Info("LockProxy starting addr store")

	p.Spawn("remoteAddrStore", func(ctx context.Context) error {
		err := p.remoteAddrStore.Start(ctx, func() {
			addrStoreStarted <- struct{}{}
		})
		if err != nil {
			p.logger.WithError(err).Info("LockProxy addr store error")
			return err
		}
		return nil
	})

	select {
	case <-addrStoreStarted:
	case <-p.ctx.Done():
		return p.ctx.Err()
	}

	p.logger.Info("LockProxy addr store started")

	p.Spawn("healthServer", func(ctx context.Context) error {
		p.logger.WithField("listenAddr", p.config.HealthListenAddr).Info("Starting health gRPC server")

		return p.healthServer.Serve(p.healthListener)
	})

	p.Spawn("proxyServer", func(ctx context.Context) error {
		p.logger.WithField("listenAddr", p.config.ProxyListenAddr).Info("Starting proxy gRPC server")

		return p.proxyServer.Serve(p.proxyListener)
	})

	p.Spawn("locker", func(ctx context.Context) error {
		return p.locker.Start(ctx)
	})

	p.Spawn("pinger", func(ctx context.Context) error {
		return p.pinger.Start(ctx)
	})

	<-p.ctx.Done()

	p.logger.Info("Shutting down")

	p.proxyServer.GracefulStop()
	p.healthServer.GracefulStop()
	_ = p.debugServer.Shutdown(context.Background())

	err := p.errGroup.Wait()
	if err != nil {
		p.logger.WithError(err).Warn("Shutdown with error")
		return err
	}

	p.logger.Info("Shutdown")

	return nil
}

func (p *LockProxy) Close() error {
	var closeErr error

	if p.debugListener != nil {
		if err := p.debugListener.Close(); err != nil {
			closeErr = multierror.Append(closeErr, xerrors.Errorf("failed to close debug listener: %w", err))
		}
	}
	if p.healthListener != nil {
		if err := p.healthListener.Close(); err != nil {
			closeErr = multierror.Append(closeErr, xerrors.Errorf("failed to close health listener: %w", err))
		}
	}
	if p.proxyListener != nil {
		if err := p.proxyListener.Close(); err != nil {
			closeErr = multierror.Append(closeErr, xerrors.Errorf("failed to close proxy listener: %w", err))
		}
	}
	if p.adapter != nil {
		if err := p.adapter.Close(); err != nil {
			closeErr = multierror.Append(closeErr, xerrors.Errorf("failed to close adapter: %w", err))
		}
	}

	return closeErr
}
