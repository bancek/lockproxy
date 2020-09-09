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

	mutex *redsync.Mutex
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
		logger:    logger.WithField("lockKey", lockKey),
	}
}

func (l *Locker) Start(ctx context.Context) error {
	l.logger.Debug("Locker creating mutex")

	l.setupMutex(ctx)

	l.logger.Info("Locker acquiring lock")

	locked, err := l.lock(ctx)
	if err != nil {
		return err
	}
	if !locked {
		return nil
	}

	l.logger.Info("Locker locked")

	ctx, cancel := context.WithCancel(ctx)
	extendErrChan := make(chan error, 1)

	go l.extendLoop(ctx, cancel, extendErrChan)

	err = l.onLocked(ctx)
	cancel()

	select {
	case extendErr := <-extendErrChan:
		err = extendErr
	default:
	}

	l.logger.Info("Locker unlocking")

	unlockErr := l.unlock()

	if err != nil {
		return xerrors.Errorf("on locked failed: %w", err)
	}

	if unlockErr != nil {
		return xerrors.Errorf("unlock failed: %w", err)
	}

	return nil
}

func (l *Locker) setupMutex(ctx context.Context) {
	pool := redigo.NewRedigoPool(l.redisPool)
	rs := redsync.New([]redsyncredis.Pool{pool})

	l.mutex = rs.NewMutex(
		l.lockKey,
		redsync.SetExpiry(time.Duration(l.lockTTL)*time.Second),
		redsync.SetRetryDelayFunc(func(tries int) time.Duration {
			// try to exit Lock as fast as possible
			if ctx.Err() != nil {
				return 0
			}
			return 500 * time.Millisecond
		}),
	)
}

func (l *Locker) lock(ctx context.Context) (locked bool, err error) {
	for {
		if err := l.mutex.Lock(); err != nil {
			if ctx.Err() != nil {
				// lock was aborted, ignore the original err
				return false, nil
			}
			if xerrors.Is(err, redsync.ErrFailed) {
				continue
			}
			return false, xerrors.Errorf("failed to lock: %w", err)
		}

		return true, nil
	}
}

func (l *Locker) extendLoop(ctx context.Context, cancel func(), extendErrChan chan error) {
	defer cancel()

	extendDelay := (time.Duration(l.lockTTL) * time.Second) / 2

	for {
		timer := time.NewTimer(extendDelay)

		select {
		case <-timer.C:
			extended, err := l.mutex.Extend()
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

			return
		}
	}
}

func (l *Locker) unlock() error {
	unlocked, err := l.mutex.Unlock()

	if err != nil {
		l.logger.WithError(err).Warn("Locker unlock failed")

		return err
	}

	if unlocked {
		l.logger.Info("Locker unlocked")
	} else {
		l.logger.Warn("Locker failed to unlock")
	}

	return err
}
