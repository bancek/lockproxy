package lockproxy

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
	"google.golang.org/grpc"
)

type LockProxy struct {
	config *Config
	logger *logrus.Entry

	ctx            context.Context
	cancel         func()
	errGroup       *errgroup.Group
	debugListener  net.Listener
	healthListener net.Listener
	proxyListener  net.Listener
	etcdClient     *clientv3.Client
	pinger         *Pinger
	addrStore      *AddrStore
	proxyDirector  *ProxyDirector
	commander      *Commander
	locker         *Locker
	debugServer    *http.Server
	healthService  *HealthService
	healthServer   *grpc.Server
	proxyServer    *grpc.Server
}

func NewLockProxy(
	config *Config,
	logger *logrus.Entry,
) *LockProxy {
	return &LockProxy{
		config: config,
		logger: logger,
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

	p.logger.WithFields(logrus.Fields{
		"endpoints": strings.Join(p.config.EtcdEndpoints, ","),
	}).Info("LockProxy connecting to etcd")

	p.etcdClient, err = NewEtcdClient(ctx, p.config)
	if err != nil {
		return xerrors.Errorf("failed to connect to etcd: %w", err)
	}

	p.logger.Info("LockProxy etcd connected")

	p.pinger = NewPinger(
		p.etcdClient,
		p.config.EtcdAddrKey,
		p.config.EtcdPingTimeout,
		p.config.EtcdPingDelay,
		p.config.EtcdPingInitialDelay,
		p.logger,
	)

	p.addrStore = NewAddrStore(p.etcdClient, p.config.EtcdAddrKey, p.logger)

	p.proxyDirector = NewProxyDirector(
		ctx,
		p.upstreamAddrProvider,
		p.config.HealthListenAddr,
		grpcDialTransportSecurity,
		p.config.ProxyRequestAbortTimeout,
		p.logger,
	)

	p.commander = NewCommander(p.config.Cmd, p.config.CmdShutdownTimeout, p.logger)

	p.locker = NewLocker(
		p.etcdClient,
		p.config.EtcdLockKey,
		p.config.EtcdLockTTL,
		p.config.EtcdUnlockTimeout,
		p.onLocked,
		p.logger,
	)

	p.debugServer = NewDebugServer()

	p.healthService = NewHealthService()
	p.healthServer = NewHealthServer(p.healthService)

	p.proxyServer = NewProxyServer(p.proxyDirector)

	return nil
}

func (p *LockProxy) upstreamAddrProvider() (addr string, isLeader bool) {
	addr = p.addrStore.Addr()
	isLeader = addr == p.config.UpstreamAddr
	return addr, isLeader
}

func (p *LockProxy) onLocked(ctx context.Context) error {
	err := p.addrStore.SetAddr(ctx, p.config.UpstreamAddr)
	if err != nil {
		return xerrors.Errorf("failed to set addr: %w", err)
	}

	return p.commander.Start(ctx)
}

func (p *LockProxy) Spawn(f func(context.Context) error) {
	p.errGroup.Go(func() error {
		err := f(p.ctx)
		if err != nil {
			p.cancel()
			return err
		}
		return nil
	})
}

func (p *LockProxy) Start() error {
	watchCreated := make(chan struct{}, 1)

	p.logger.Info("LockProxy starting watch")

	p.Spawn(func(ctx context.Context) error {
		err := p.addrStore.Watch(ctx, func() {
			watchCreated <- struct{}{}
		})
		if err != nil {
			p.logger.WithError(err).Info("LockProxy watch error")
			return err
		}
		return nil
	})

	p.logger.Info("LockProxy waiting for watch created")

	select {
	case <-watchCreated:
	case <-p.ctx.Done():
		return p.ctx.Err()
	}

	p.logger.Info("LockProxy watch created")

	err := p.addrStore.Init(p.ctx)
	if err != nil {
		return xerrors.Errorf("failed to initialize addr store: %w", err)
	}

	p.logger.Info("LockProxy store initialized")

	p.Spawn(func(ctx context.Context) error {
		p.logger.WithField("listenAddr", p.config.DebugListenAddr).Info("Starting debug HTTP server")

		err := p.debugServer.Serve(p.debugListener)
		if ctx.Err() == nil {
			return err
		}
		return nil
	})

	p.Spawn(func(ctx context.Context) error {
		p.logger.WithField("listenAddr", p.config.HealthListenAddr).Info("Starting health gRPC server")

		return p.healthServer.Serve(p.healthListener)
	})

	p.Spawn(func(ctx context.Context) error {
		p.logger.WithField("listenAddr", p.config.ProxyListenAddr).Info("Starting proxy gRPC server")

		return p.proxyServer.Serve(p.proxyListener)
	})

	p.Spawn(func(ctx context.Context) error {
		return p.locker.Start(ctx)
	})

	p.Spawn(func(ctx context.Context) error {
		return p.pinger.Start(ctx)
	})

	<-p.ctx.Done()

	p.logger.Info("Shutting down")

	p.proxyServer.GracefulStop()
	p.healthServer.GracefulStop()
	_ = p.debugServer.Shutdown(context.Background())

	err = p.errGroup.Wait()
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
	if p.etcdClient != nil {
		if err := p.etcdClient.Close(); err != nil {
			closeErr = multierror.Append(closeErr, xerrors.Errorf("failed to close etcd client: %w", err))
		}
	}

	return closeErr
}
