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

	// RedisMaxIdle is the max amount of idle connections to Redis.
	// (LOCKPROXY_REDISMAXIDLE)
	RedisMaxIdle int `default:"10"`

	// RedisIdleTimeout is the duration after which the connections remaining idle
	// will be closed.
	// (LOCKPROXY_REDISIDLETIMEOUT)
	RedisIdleTimeout time.Duration `default:"5m"`

	// RedisLockTTL is the duration of the redis lock (in seconds). The lock will
	// be refreshed every RedisLockTTL seconds.
	// (LOCKPROXY_REDISLOCKTTL)
	RedisLockTTL int `default:"10"`

	// RedisLockKey is the redis key used for redis lock.
	// (LOCKPROXY_REDISLOCKKEY)
	RedisLockKey string `required:"true"`

	// RedisAddrKey is the redis key used to store the address of the current
	// leader.
	// (LOCKPROXY_REDISADDRKEY)
	RedisAddrKey string `required:"true"`

	// RedisPingTimeout is the timeout after which the ping is deemed failed and the
	// process will exit.
	// (LOCKPROXY_REDISPINGTIMEOUT)
	RedisPingTimeout time.Duration `default:"10s"`

	// RedisPingDelay is the delay between pings.
	// (LOCKPROXY_REDISPINGDELAY)
	RedisPingDelay time.Duration `default:"10s"`

	// RedisPingInitialDelay is the delay before the first ping.
	// (LOCKPROXY_REDISPINGINITIALDELAY)
	RedisPingInitialDelay time.Duration `default:"10s"`
}
