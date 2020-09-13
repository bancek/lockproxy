package redisadapter

import (
	"context"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"

	"github.com/bancek/lockproxy/pkg/lockproxy"
)

type RedisAdapter struct {
	config *RedisConfig
	logger *logrus.Entry

	redisDialer RedisDialer
	redisPool   *redis.Pool
}

func NewRedisAdapter(config *RedisConfig, logger *logrus.Entry) *RedisAdapter {
	return &RedisAdapter{
		config: config,
		logger: logger,
	}
}

func (a *RedisAdapter) Init(ctx context.Context) error {
	safeRedisURL := GetSafeURL(a.config.RedisURL)

	a.logger.WithFields(logrus.Fields{
		"redisURL": safeRedisURL,
	}).Info("RedisAdapter connecting to redis")

	redisDialer := NewRedisDialer(a.config)
	redisPool := NewRedisPool(redisDialer, a.config)

	conn := redisPool.Get()
	defer conn.Close()
	if _, err := conn.Do("PING"); err != nil {
		return xerrors.Errorf("failed to connect to redis: %s: %w", safeRedisURL, err)
	}

	a.redisDialer = redisDialer
	a.redisPool = redisPool

	a.logger.Info("RedisAdapter redis connected")

	return nil
}

func (a *RedisAdapter) GetLocker(onLocked func(ctx context.Context) error) (lockproxy.Locker, error) {
	if a.redisDialer == nil || a.redisPool == nil {
		return nil, xerrors.Errorf("not initialized")
	}
	return NewLocker(
		a.redisPool,
		a.config.RedisLockKey,
		a.config.RedisLockTTL,
		onLocked,
		a.logger,
	), nil
}

func (a *RedisAdapter) GetRemoteAddrStore(localAddrStore lockproxy.LocalAddrStore) (lockproxy.RemoteAddrStore, error) {
	if a.redisDialer == nil || a.redisPool == nil {
		return nil, xerrors.Errorf("not initialized")
	}
	return NewAddrStore(
		localAddrStore,
		a.redisDialer,
		a.redisPool,
		a.config.RedisAddrKey,
		a.config.RedisRetryInitialInterval,
		a.config.RedisRetryMaxElapsedTime,
		a.logger,
	), nil
}

func (a *RedisAdapter) Close() error {
	if a.redisPool != nil {
		if err := a.redisPool.Close(); err != nil {
			return xerrors.Errorf("failed to close redis pool: %w", err)
		}
	}
	return nil
}
