package redisadapter

import (
	"context"
	"sync"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

// AddrStore stores the current proxy address locally.
// It sets the address in redis and watches the changes.
type AddrStore struct {
	redisDialer RedisDialer
	redisPool   *redis.Pool
	addrKey     string
	logger      *logrus.Entry

	addr  string
	mutex sync.RWMutex
}

func NewAddrStore(
	redisDialer RedisDialer,
	redisPool *redis.Pool,
	addrKey string,
	logger *logrus.Entry,
) *AddrStore {
	return &AddrStore{
		redisDialer: redisDialer,
		redisPool:   redisPool,
		addrKey:     addrKey,
		logger:      logger.WithField("addrKey", addrKey),

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

	conn, err := s.redisPool.GetContext(ctx)
	if err != nil {
		return xerrors.Errorf("failed to get redis conn: %w", err)
	}
	defer conn.Close()

	addr, err := redis.String(conn.Do("GET", s.addrKey))
	if err != nil && err != redis.ErrNil {
		return xerrors.Errorf("failed to get addr: %s: %w", s.addrKey, err)
	}

	s.setAddr(addr)

	s.logger.WithFields(logrus.Fields{
		"addr": addr,
	}).Debug("AddrStore loaded initial addr")

	return nil
}

func (s *AddrStore) Watch(ctx context.Context, onCreated func()) error {
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
			if onCreated != nil {
				onCreated()
			}

		case redis.Message:
			addr := string(msg.Data)

			s.logger.WithFields(logrus.Fields{
				"addr": addr,
			}).Debug("AddrStore receive message")

			s.setAddr(addr)

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
