package redisadapter

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"

	"github.com/bancek/lockproxy/pkg/lockproxy"
)

// AddrStore stores the current proxy address locally.
// It sets the address in redis and watches the changes.
type AddrStore struct {
	localAddrStore       lockproxy.LocalAddrStore
	redisDialer          RedisDialer
	redisPool            *redis.Pool
	addrKey              string
	retryInitialInterval time.Duration
	retryMaxElapsedTime  time.Duration
	logger               *logrus.Entry
}

func NewAddrStore(
	localAddrStore lockproxy.LocalAddrStore,
	redisDialer RedisDialer,
	redisPool *redis.Pool,
	addrKey string,
	retryInitialInterval time.Duration,
	retryMaxElapsedTime time.Duration,
	logger *logrus.Entry,
) *AddrStore {
	return &AddrStore{
		localAddrStore:       localAddrStore,
		redisDialer:          redisDialer,
		redisPool:            redisPool,
		addrKey:              addrKey,
		retryInitialInterval: retryInitialInterval,
		retryMaxElapsedTime:  retryMaxElapsedTime,
		logger:               logger.WithField("addrKey", addrKey),
	}
}

func (s *AddrStore) Addr(ctx context.Context) string {
	return s.localAddrStore.Addr(ctx)
}

func (s *AddrStore) Refresh(ctx context.Context) (addr string, err error) {
	err = backoff.RetryNotify(
		func() (err error) {
			addr, err = s.refresh(ctx)
			return err
		},
		backoff.WithContext(lockproxy.GetExponentialBackOff(s.retryInitialInterval, s.retryMaxElapsedTime), ctx),
		func(err error, next time.Duration) {
			s.logger.WithFields(logrus.Fields{
				"nextRetryIn":   next,
				logrus.ErrorKey: err,
			}).Warn("AddrStore refresh error")
		},
	)
	if err != nil {
		return "", err
	}
	return addr, nil
}

func (s *AddrStore) refresh(ctx context.Context) (addr string, err error) {
	conn, err := s.redisPool.GetContext(ctx)
	if err != nil {
		return "", xerrors.Errorf("failed to get redis conn: %w", err)
	}
	defer conn.Close()

	addr, err = redis.String(conn.Do("GET", s.addrKey))
	if err != nil && err != redis.ErrNil {
		return "", xerrors.Errorf("failed to get addr: %s: %w", s.addrKey, err)
	}

	s.localAddrStore.SetAddr(ctx, addr)

	return addr, nil
}

func (s *AddrStore) init(ctx context.Context) error {
	s.logger.Debug("AddrStore loading initial addr")

	addr, err := s.refresh(ctx)
	if err != nil {
		return xerrors.Errorf("refresh failed: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"addr": addr,
	}).Debug("AddrStore loaded initial addr")

	return nil
}

func (s *AddrStore) Start(ctx context.Context, onStarted func()) error {
	s.logger.Debug("AddrStore starting watching")

	onStartedCalled := false

	for {
		err := backoff.RetryNotify(
			func() error {
				err := s.start(ctx, func() {
					if !onStartedCalled {
						onStartedCalled = true
						onStarted()
					}
				})
				if err != nil {
					return xerrors.Errorf("failed to start addr store: %w", err)
				}
				return nil
			},
			backoff.WithContext(lockproxy.GetExponentialBackOff(s.retryInitialInterval, s.retryMaxElapsedTime), ctx),
			func(err error, next time.Duration) {
				s.logger.WithFields(logrus.Fields{
					"nextRetryIn":   next,
					logrus.ErrorKey: err,
				}).Warn("AddrStore watch error")
			},
		)
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		if err != nil {
			s.logger.WithError(err).Warn("AddrStore watch error")
		}
	}
}

func (s *AddrStore) start(ctx context.Context, onStarted func()) error {
	s.logger.Debug("AddrStore starting watching")
	redisConn, err := s.redisDialer()
	if err != nil {
		return xerrors.Errorf("failed to get redis conn: %w", err)
	}
	defer redisConn.Close()

	conn := redis.PubSubConn{Conn: redisConn}

	done := make(chan struct{})

	go func() {
		select {
		case <-done:
			return
		case <-ctx.Done():
			// we need to close the connection for conn.Receive() to stop
			redisConn.Close()
		}
	}()

	err = conn.Subscribe(s.addrKey)
	if err != nil {
		return xerrors.Errorf("failed to subscribe: %s: %w", s.addrKey, err)
	}

	for {
		switch msg := conn.Receive().(type) {
		case redis.Subscription:
			s.logger.Debug("AddrStore subscribed")

			err := s.init(ctx)
			if err != nil {
				return xerrors.Errorf("failed to init: %w", err)
			}

			if onStarted != nil {
				onStarted()
			}

		case redis.Message:
			addr := string(msg.Data)

			s.logger.WithFields(logrus.Fields{
				"addr": addr,
			}).Debug("AddrStore receive message")

			s.localAddrStore.SetAddr(ctx, addr)

		case redis.Pong:

		default:
			select {
			case <-ctx.Done():
				return nil
			default:
			}

			err := msg.(error)

			s.logger.WithError(err).Debug("AddrStore receive error")

			close(done)

			return err
		}
	}
}

func (s *AddrStore) SetAddr(ctx context.Context, addr string) error {
	s.logger.WithFields(logrus.Fields{
		"addr": addr,
	}).Debug("AddrStore setting addr remotely")

	return backoff.RetryNotify(
		func() error {
			err := s.setAddr(ctx, addr)
			if err != nil {
				return xerrors.Errorf("failed to set addr: %w", err)
			}
			return nil
		},
		backoff.WithContext(lockproxy.GetExponentialBackOff(s.retryInitialInterval, s.retryMaxElapsedTime), ctx),
		func(err error, next time.Duration) {
			s.logger.WithFields(logrus.Fields{
				"nextRetryIn":   next,
				logrus.ErrorKey: err,
			}).Warn("AddrStore set addr error")
		},
	)
}

func (s *AddrStore) setAddr(ctx context.Context, addr string) error {
	conn, err := s.redisPool.GetContext(ctx)
	if err != nil {
		return xerrors.Errorf("failed to get redis conn: %w", err)
	}
	defer conn.Close()

	_, err = conn.Do("SET", s.addrKey, addr)
	if err != nil {
		return xerrors.Errorf("redis set error: %w", err)
	}

	_, err = conn.Do("PUBLISH", s.addrKey, addr)
	if err != nil {
		return xerrors.Errorf("redis publish error: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"addr": addr,
	}).Debug("AddrStore setting addr remotely done")

	return nil
}

var _ lockproxy.RemoteAddrStore = (*AddrStore)(nil)
