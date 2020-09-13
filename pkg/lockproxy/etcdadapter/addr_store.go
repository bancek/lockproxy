package etcdadapter

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
	"golang.org/x/xerrors"

	"github.com/bancek/lockproxy/pkg/lockproxy"
)

// AddrStore stores the current proxy address locally.
// It sets the address in etcd and watches the changes.
type AddrStore struct {
	localAddrStore lockproxy.LocalAddrStore
	etcdClient     *clientv3.Client
	addrKey        string
	logger         *logrus.Entry
}

func NewAddrStore(
	localAddrStore lockproxy.LocalAddrStore,
	etcdClient *clientv3.Client,
	addrKey string,
	logger *logrus.Entry,
) *AddrStore {
	return &AddrStore{
		localAddrStore: localAddrStore,
		etcdClient:     etcdClient,
		addrKey:        addrKey,
		logger:         logger.WithField("addrKey", addrKey),
	}
}

func (s *AddrStore) Addr(ctx context.Context) string {
	return s.localAddrStore.Addr(ctx)
}

func (s *AddrStore) Refresh(ctx context.Context) (addr string, err error) {
	resp, err := s.etcdClient.Get(ctx, s.addrKey)
	if err != nil {
		return "", xerrors.Errorf("failed to get addr: %s: %w", s.addrKey, err)
	}

	// if the value is not set, resp.Kvs will be nil
	for _, kv := range resp.Kvs {
		addr = string(kv.Value)
	}

	s.localAddrStore.SetAddr(ctx, addr)

	return addr, nil
}

func (s *AddrStore) init(ctx context.Context) error {
	s.logger.Debug("AddrStore loading initial addr")

	addr, err := s.Refresh(ctx)
	if err != nil {
		return err
	}

	s.logger.WithFields(logrus.Fields{
		"addr": addr,
	}).Debug("AddrStore loaded initial addr")

	return nil
}

func (s *AddrStore) Start(ctx context.Context, onStarted func()) error {
	s.logger.Debug("AddrStore starting watching")

	ctx = clientv3.WithRequireLeader(ctx)

	rch := s.etcdClient.Watch(ctx, s.addrKey, clientv3.WithCreatedNotify())

	for wresp := range rch {
		if wresp.Canceled {
			s.logger.Debug("AddrStore watch canceled")
			return nil
		} else if wresp.Created {
			s.logger.Debug("AddrStore watch created")

			err := s.init(ctx)
			if err != nil {
				return xerrors.Errorf("failed to init: %w", err)
			}

			if onStarted != nil {
				onStarted()
			}
		} else if err := wresp.Err(); err != nil {
			s.logger.WithError(err).Debug("AddrStore watch error")
			return err
		} else if len(wresp.Events) > 0 {
			for _, ev := range wresp.Events {
				s.logger.WithFields(logrus.Fields{
					"eventType":  ev.Type,
					"eventKey":   string(ev.Kv.Key),
					"eventValue": string(ev.Kv.Value),
				}).Debug("AddrStore watch event")

				if ev.Type == clientv3.EventTypePut {
					s.localAddrStore.SetAddr(ctx, string(ev.Kv.Value))
				}
			}
		}
	}

	return nil
}

func (s *AddrStore) SetAddr(ctx context.Context, addr string) error {
	s.logger.WithFields(logrus.Fields{
		"addr": addr,
	}).Debug("AddrStore setting addr remotely")

	_, err := s.etcdClient.Put(ctx, s.addrKey, addr)
	if err != nil {
		return xerrors.Errorf("etcd put error: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"addr": addr,
	}).Debug("AddrStore setting addr remotely done")

	return nil
}

var _ lockproxy.RemoteAddrStore = (*AddrStore)(nil)
