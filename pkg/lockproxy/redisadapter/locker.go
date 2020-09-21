package redisadapter

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/go-redsync/redsync/v3"
	redsyncredis "github.com/go-redsync/redsync/v3/redis"
	"github.com/go-redsync/redsync/v3/redis/redigo"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"

	"github.com/bancek/lockproxy/pkg/lockproxy"
)

type Locker struct {
	redisPool            *redis.Pool
	lockKey              string
	lockTTL              int
	lockRetryDelay       time.Duration
	unlockTimeout        time.Duration
	retryInitialInterval time.Duration
	retryMaxElapsedTime  time.Duration
	onLocked             func(ctx context.Context) error
	logger               *logrus.Entry

	mutex *redsync.Mutex
}

func NewLocker(
	redisPool *redis.Pool,
	lockKey string,
	lockTTL int,
	lockRetryDelay time.Duration,
	unlockTimeout time.Duration,
	retryInitialInterval time.Duration,
	retryMaxElapsedTime time.Duration,
	onLocked func(ctx context.Context) error,
	logger *logrus.Entry,
) *Locker {
	return &Locker{
		redisPool:            redisPool,
		lockKey:              lockKey,
		lockTTL:              lockTTL,
		lockRetryDelay:       lockRetryDelay,
		unlockTimeout:        unlockTimeout,
		retryInitialInterval: retryInitialInterval,
		retryMaxElapsedTime:  retryMaxElapsedTime,
		onLocked:             onLocked,
		logger:               logger.WithField("lockKey", lockKey),
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
		return xerrors.Errorf("unlock failed: %w", unlockErr)
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
			return l.lockRetryDelay
		}),
	)
}

func (l *Locker) lock(ctx context.Context) (locked bool, err error) {
	for {
		if err := l.mutexLock(ctx); err != nil {
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
			extended, err := l.mutexExtend(ctx)
			if err != nil {
				err = xerrors.Errorf("failed to extend the lock: %w", err)
				l.logger.WithError(err).Warn("Locker extend loop mutex extend failed")
				extendErrChan <- err
				return
			}
			if !extended {
				err = xerrors.Errorf("lock not extended")
				l.logger.WithError(err).Warn("Locker ectend loop mutex not extended")
				extendErrChan <- err
				return
			}

		case <-ctx.Done():
			l.logger.WithError(ctx.Err()).Info("Locker extend loop done")
			timer.Stop()

			return
		}
	}
}

func (l *Locker) unlock() error {
	// we cannot use ctx for unlock because it might already be cancelled, but we
	// should still limit it.
	ctx, cancel := context.WithTimeout(context.Background(), l.unlockTimeout)
	defer cancel()

	unlocked, err := l.mutexUnlock(ctx)

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

func (l *Locker) mutexLock(ctx context.Context) error {
	return backoff.RetryNotify(
		func() error {
			err := l.mutex.Lock()
			if xerrors.Is(err, redsync.ErrFailed) {
				// retry ErrFailed right away, without backoff
				return &backoff.PermanentError{
					Err: err,
				}
			}
			return err
		},
		backoff.WithContext(lockproxy.GetExponentialBackOff(l.retryInitialInterval, l.retryMaxElapsedTime), ctx),
		func(err error, next time.Duration) {
			l.logger.WithFields(logrus.Fields{
				"nextRetryIn":   next,
				logrus.ErrorKey: err,
			}).Warn("Locker mutex lock error")
		},
	)
}

func (l *Locker) mutexExtend(ctx context.Context) (extended bool, err error) {
	err = backoff.RetryNotify(
		func() (err error) {
			extended, err = l.mutex.Extend()
			return err
		},
		backoff.WithContext(lockproxy.GetExponentialBackOff(l.retryInitialInterval, l.retryMaxElapsedTime), ctx),
		func(err error, next time.Duration) {
			l.logger.WithFields(logrus.Fields{
				"nextRetryIn":   next,
				logrus.ErrorKey: err,
			}).Warn("Locker mutex extend error")
		},
	)
	if err != nil {
		return false, err
	}
	return extended, nil
}

func (l *Locker) mutexUnlock(ctx context.Context) (unlocked bool, err error) {
	err = backoff.RetryNotify(
		func() (err error) {
			unlocked, err = l.mutex.Unlock()
			return err
		},
		backoff.WithContext(lockproxy.GetExponentialBackOff(l.retryInitialInterval, l.retryMaxElapsedTime), ctx),
		func(err error, next time.Duration) {
			l.logger.WithFields(logrus.Fields{
				"nextRetryIn":   next,
				logrus.ErrorKey: err,
			}).Warn("Locker mutex extend error")
		},
	)
	if err != nil {
		return false, err
	}
	return unlocked, nil
}
