package lockproxy

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/concurrency"
	"golang.org/x/xerrors"
)

type Locker struct {
	etcdClient        *clientv3.Client
	lockKey           string
	lockTTL           int
	etcdUnlockTimeout time.Duration
	onLocked          func(ctx context.Context) error
	logger            *logrus.Entry
}

func NewLocker(
	etcdClient *clientv3.Client,
	lockKey string,
	lockTTL int,
	etcdUnlockTimeout time.Duration,
	onLocked func(ctx context.Context) error,
	logger *logrus.Entry,
) *Locker {
	return &Locker{
		etcdClient:        etcdClient,
		lockKey:           lockKey,
		lockTTL:           lockTTL,
		etcdUnlockTimeout: etcdUnlockTimeout,
		onLocked:          onLocked,
		logger:            logger,
	}
}

func (l *Locker) Start(ctx context.Context) error {
	l.logger.WithField("lockKey", l.lockKey).Debug("Locker creating etcd concurrency session")

	s, err := concurrency.NewSession(l.etcdClient, concurrency.WithTTL(l.lockTTL))
	if err != nil {
		return xerrors.Errorf("failed to create etcd concurrency session: %w", err)
	}

	l.logger.WithField("lockKey", l.lockKey).Debug("Locker creating mutex")

	m := concurrency.NewMutex(s, l.lockKey)

	l.logger.WithField("lockKey", l.lockKey).Info("Locker acquiring lock")

	if err := m.Lock(ctx); err != nil {
		return xerrors.Errorf("failed to lock: %w", err)
	}

	l.logger.WithField("lockKey", l.lockKey).Info("Locker locked")

	err = l.onLocked(ctx)

	// we cannot use ctx for Unlock because it might already be cancelled,
	// but we should still limit it.
	unlockCtx, unlockCtxCancel := context.WithTimeout(context.Background(), l.etcdUnlockTimeout)
	defer unlockCtxCancel()

	unlockErr := m.Unlock(unlockCtx)

	if unlockErr == nil {
		l.logger.WithField("lockKey", l.lockKey).Info("Locker unlocked")
	} else {
		l.logger.WithFields(logrus.Fields{
			"lockKey": l.lockKey,
		}).Warn("Locker unlock failed")
	}

	if err != nil {
		return xerrors.Errorf("on locked failed: %w", err)
	}

	if unlockErr != nil {
		return xerrors.Errorf("unlock failed: %w", err)
	}

	return nil
}
