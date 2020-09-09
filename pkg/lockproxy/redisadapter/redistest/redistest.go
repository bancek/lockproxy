package redistest

import (
	"os"
	"testing"
	"time"

	"github.com/gomodule/redigo/redis"
	. "github.com/onsi/gomega"

	"github.com/bancek/lockproxy/pkg/lockproxy/redisadapter"
	"github.com/bancek/tcpproxy"
)

var RedisAddr string
var RedisProxy *tcpproxy.Proxy
var RedisDialer redisadapter.RedisDialer
var RedisPool *redis.Pool

func RedisInitTesting(t *testing.T) bool {
	RedisAddr = os.Getenv("REDIS_ADDR")
	if RedisAddr == "" {
		t.Skip("Missing REDIS_ADDR env variable")
		return false
	}

	return true
}

func RedisBeforeEach() {
	var err error
	RedisProxy, err = tcpproxy.NewUnusedAddr("127.0.0.1", RedisAddr)
	Expect(err).NotTo(HaveOccurred())
	Expect(RedisProxy.Start()).To(Succeed())

	cfg := &redisadapter.RedisConfig{
		RedisURL:               "redis://" + RedisProxy.ListenAddr(),
		RedisDialTimeout:       10 * time.Second,
		RedisDialKeepAliveTime: 1 * time.Minute,
		RedisMaxIdle:           10,
		RedisIdleTimeout:       5 * time.Minute,
	}

	RedisDialer = redisadapter.NewRedisDialer(cfg)

	RedisPool = redisadapter.NewRedisPool(RedisDialer, cfg)
}

func RedisAfterEach() {
	RedisPool.Close()
	RedisPool = nil

	RedisProxy.Close()
	RedisProxy = nil
}
