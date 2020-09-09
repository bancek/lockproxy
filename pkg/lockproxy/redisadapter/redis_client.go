package redisadapter

import (
	"net/url"
	"time"

	"github.com/gomodule/redigo/redis"
)

type RedisDialer func() (redis.Conn, error)

func RedisDial(config *RedisConfig) (redis.Conn, error) {
	c, err := redis.DialURL(
		config.RedisURL,
		redis.DialConnectTimeout(config.RedisDialTimeout),
		redis.DialKeepAlive(config.RedisDialKeepAliveTime),
	)
	if err != nil {
		return nil, err
	}
	return c, err
}

func NewRedisDialer(config *RedisConfig) RedisDialer {
	return func() (redis.Conn, error) {
		return RedisDial(config)
	}
}

func NewRedisPool(dial RedisDialer, config *RedisConfig) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     config.RedisMaxIdle,
		IdleTimeout: config.RedisIdleTimeout,
		Dial:        dial,
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
}

func GetSafeURL(redisURL string) string {
	u, err := url.Parse(redisURL)
	if err != nil {
		return redisURL
	}
	if _, ok := u.User.Password(); ok {
		u.User = url.UserPassword(u.User.Username(), "XXXXXXXX")
	}
	return u.String()
}
