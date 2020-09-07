package etcdadapter

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
	"golang.org/x/xerrors"
)

// AddrStore stores the current proxy address locally.
// It sets the address in etcd and watches the changes.
type AddrStore struct {
	etcdClient *clientv3.Client
	addrKey    string
	logger     *logrus.Entry

	addr  string
	mutex sync.RWMutex
}

func NewAddrStore(
	etcdClient *clientv3.Client,
	addrKey string,
	logger *logrus.Entry,
) *AddrStore {
	return &AddrStore{
		etcdClient: etcdClient,
		addrKey:    addrKey,
		logger:     logger.WithField("addrKey", addrKey),

		addr: "",
	}
}

func (s *AddrStore) Addr() string {
	s.mutex.RLock()
	addr := s.addr
	s.mutex.RUnlock()
	return addr
}

func (s *AddrStore) setAddr(addr string) {
	s.logger.WithFields(logrus.Fields{
		"addr": addr,
	}).Debug("AddrStore setting addr locally")

	s.mutex.Lock()
	s.addr = addr
	s.mutex.Unlock()
}

func (s *AddrStore) Init(ctx context.Context) error {
	s.logger.Debug("AddrStore loading initial addr")

	resp, err := s.etcdClient.Get(ctx, s.addrKey)
	if err != nil {
		return xerrors.Errorf("failed to get addr: %s: %w", s.addrKey, err)
	}

	addr := ""

	// if the value is not set, resp.Kvs will be nil
	for _, kv := range resp.Kvs {
		addr = string(kv.Value)
	}

	s.setAddr(addr)

	s.logger.WithFields(logrus.Fields{
		"addr": addr,
	}).Debug("AddrStore loaded initial addr")

	return nil
}

func (s *AddrStore) Watch(ctx context.Context, onCreated func()) error {
	s.logger.Debug("AddrStore starting watching")

	ctx = clientv3.WithRequireLeader(ctx)

	rch := s.etcdClient.Watch(ctx, s.addrKey, clientv3.WithCreatedNotify())

	for wresp := range rch {
		if wresp.Canceled {
			s.logger.Debug("AddrStore watch canceled")
			return nil
		} else if wresp.Created {
			s.logger.Debug("AddrStore watch created")
			if onCreated != nil {
				onCreated()
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
					s.setAddr(string(ev.Kv.Value))
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
