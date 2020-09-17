package redisadapter

import "time"

type RedisConfig struct {
	// RedisURL is an address of a Redis server.
	// (LOCKPROXY_REDISURL)
	RedisURL string `default:"redis://localhost:6379"`

	// RedisDialTimeout is the timeout for failing to establish a redis
	// connection.
	// (LOCKPROXY_REDISDIALTIMEOUT)
	RedisDialTimeout time.Duration `default:"10s"`

	// RedisDialKeepAliveTime is the time after which client pings the server to
	// see if the transport is alive.
	// (LOCKPROXY_REDISDIALKEEPALIVETIME)
	RedisDialKeepAliveTime time.Duration `default:"10s"`

	// RedisDialClientName is the name of the Redis client (useful for debugging
	// with Redis CLIENT LIST command).
	// (LOCKPROXY_REDISDIALCLIENTNAME)
	RedisDialClientName string `default:"lockproxy"`

	// RedisMaxIdle is the max amount of idle connections to Redis.
	// (LOCKPROXY_REDISMAXIDLE)
	RedisMaxIdle int `default:"10"`

	// RedisIdleTimeout is the duration after which the connections remaining idle
	// will be closed.
	// (LOCKPROXY_REDISIDLETIMEOUT)
	RedisIdleTimeout time.Duration `default:"5m"`

	// RedisLockTTL is the duration of the Redis lock (in seconds). The lock will
	// be refreshed every RedisLockTTL seconds.
	// (LOCKPROXY_REDISLOCKTTL)
	RedisLockTTL int `default:"10"`

	// RedisLockTTL is the duration of the Redis lock (in seconds). The lock will
	// be refreshed every RedisLockTTL seconds.
	// (LOCKPROXY_REDISLOCKTTL)
	RedisLockRetryDelay time.Duration `default:"100ms"`

	// RedisUnlockTimeout is the max duration for waiting etcd to unlock after the
	// Cmd is stopped.
	// (LOCKPROXY_REDISUNLOCKTIMEOUT)
	RedisUnlockTimeout time.Duration `default:"10s"`

	// RedisLockKey is the Redis key used for redis lock.
	// (LOCKPROXY_REDISLOCKKEY)
	RedisLockKey string `required:"true"`

	// RedisAddrKey is the Redis key used to store the address of the current
	// leader.
	// (LOCKPROXY_REDISADDRKEY)
	RedisAddrKey string `required:"true"`

	// RedisRetryInitialInterval is the initial delay for retrying.
	// (LOCKPROXY_REDISRETRYINITIALINTERVAL)
	RedisRetryInitialInterval time.Duration `default:"100ms"`

	// RedisRetryMaxElapsedTime is the max elapsed time for retrying.
	// (LOCKPROXY_REDISRETRYMAXELAPSEDTIME)
	RedisRetryMaxElapsedTime time.Duration `default:"10s"`
}
