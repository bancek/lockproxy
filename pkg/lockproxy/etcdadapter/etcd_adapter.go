package etcdadapter

import (
	"context"
	"strings"

	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
	"golang.org/x/xerrors"

	"github.com/bancek/lockproxy/pkg/lockproxy"
)

type EtcdAdapter struct {
	config *EtcdConfig
	logger *logrus.Entry

	etcdClient *clientv3.Client
}

func NewEtcdAdapter(config *EtcdConfig, logger *logrus.Entry) *EtcdAdapter {
	return &EtcdAdapter{
		config: config,
		logger: logger,
	}
}

func (a *EtcdAdapter) Init(ctx context.Context) error {
	a.logger.WithFields(logrus.Fields{
		"endpoints": strings.Join(a.config.EtcdEndpoints, ","),
	}).Info("EtcdAdapter connecting to etcd")

	etcdClient, err := NewEtcdClient(ctx, a.config)
	if err != nil {
		return xerrors.Errorf("failed to connect to etcd: %w", err)
	}
	a.etcdClient = etcdClient

	a.logger.Info("EtcdAdapter etcd connected")

	return nil
}

func (a *EtcdAdapter) GetLocker(onLocked func(ctx context.Context) error) (lockproxy.Locker, error) {
	if a.etcdClient == nil {
		return nil, xerrors.Errorf("not initialized")
	}
	return NewLocker(
		a.etcdClient,
		a.config.EtcdLockKey,
		a.config.EtcdLockTTL,
		a.config.EtcdUnlockTimeout,
		onLocked,
		a.logger,
	), nil
}

func (a *EtcdAdapter) GetRemoteAddrStore(localAddrStore lockproxy.LocalAddrStore) (lockproxy.RemoteAddrStore, error) {
	if a.etcdClient == nil {
		return nil, xerrors.Errorf("not initialized")
	}
	return NewAddrStore(localAddrStore, a.etcdClient, a.config.EtcdAddrKey, a.logger), nil
}

func (a *EtcdAdapter) Close() error {
	if a.etcdClient != nil {
		if err := a.etcdClient.Close(); err != nil {
			return xerrors.Errorf("failed to close etcd client: %w", err)
		}
	}
	return nil
}
