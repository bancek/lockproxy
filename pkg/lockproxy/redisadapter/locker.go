package redisadapter

import (
	"context"
	"time"

	"github.com/go-redsync/redsync/v3"
	redsyncredis "github.com/go-redsync/redsync/v3/redis"
	"github.com/go-redsync/redsync/v3/redis/redigo"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

type Locker struct {
	redisPool *redis.Pool
	lockKey   string
	lockTTL   int
	onLocked  func(ctx context.Context) error
	logger    *logrus.Entry
}

func NewLocker(
	redisPool *redis.Pool,
	lockKey string,
	lockTTL int,
	onLocked func(ctx context.Context) error,
	logger *logrus.Entry,
) *Locker {
	return &Locker{
		redisPool: redisPool,
		lockKey:   lockKey,
		lockTTL:   lockTTL,
		onLocked:  onLocked,
		logger:    logger,
	}
}

func (l *Locker) Start(ctx context.Context) error {
	pool := redigo.NewRedigoPool(l.redisPool)
	rs := redsync.New([]redsyncredis.Pool{pool})

	l.logger.WithField("lockKey", l.lockKey).Debug("Locker creating mutex")

	mutex := rs.NewMutex(l.lockKey, redsync.SetExpiry(time.Duration(l.lockTTL)*time.Second))

	l.logger.WithField("lockKey", l.lockKey).Info("Locker acquiring lock")

	for {
		if err := mutex.Lock(); err != nil {
			if xerrors.Is(err, redsync.ErrFailed) {
				continue
			}
			return xerrors.Errorf("failed to lock: %w", err)
		}
		break
	}

	l.logger.WithField("lockKey", l.lockKey).Info("Locker locked")

	ctx, cancel := context.WithCancel(ctx)
	extendErrChan := make(chan error, 1)

	go func() {
		extendDelay := (time.Duration(l.lockTTL) * time.Second) / 2

		for {
			timer := time.NewTimer(extendDelay)

			select {
			case <-timer.C:
				extended, err := mutex.Extend()
				if err != nil {
					extendErrChan <- xerrors.Errorf("failed to extend the lock: %w", err)
					return
				}
				if !extended {
					extendErrChan <- xerrors.Errorf("lock not extended")
					return
				}

			case <-ctx.Done():
				timer.Stop()
				cancel()

				return
			}
		}
	}()

	err := l.onLocked(ctx)

	select {
	case extendErr := <-extendErrChan:
		err = extendErr
	default:
	}

	unlocked, unlockErr := mutex.Unlock()

	if unlockErr == nil {
		if unlocked {
			l.logger.WithField("lockKey", l.lockKey).Info("Locker unlocked")
		} else {
			l.logger.WithField("lockKey", l.lockKey).Warn("Locker failed to unlock")
		}
	} else {
		l.logger.WithFields(logrus.Fields{
			"lockKey": l.lockKey,
		}).WithError(unlockErr).Warn("Locker unlock failed")
	}

	if err != nil {
		return xerrors.Errorf("on locked failed: %w", err)
	}

	if unlockErr != nil {
		return xerrors.Errorf("unlock failed: %w", err)
	}

	return nil
}
