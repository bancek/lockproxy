package lockproxy_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	. "github.com/onsi/gomega"

	. "github.com/bancek/lockproxy/pkg/lockproxy"
	"github.com/bancek/lockproxy/pkg/lockproxy/etcdadapter"
	"github.com/bancek/lockproxy/pkg/lockproxy/etcdadapter/etcdtest"
	"github.com/bancek/lockproxy/pkg/lockproxy/redisadapter"
	"github.com/bancek/lockproxy/pkg/lockproxy/redisadapter/redistest"
	"github.com/bancek/lockproxy/pkg/lockproxy/testhelpers"
)

type AdapterTest interface {
	Name() string
	SetupEnv(envPrefix string)
	SetLockTTL(envPrefix string, ttlSeconds int)
	SetUnlockTimeout(envPrefix string, timeout time.Duration)
	GetAdapter(envPrefix string) Adapter
	Abort()
	Resume()
}

func BuildAdapter(ctx context.Context, t AdapterTest) Adapter {
	adapter := t.GetAdapter(EnvPrefix)
	Expect(adapter.Init(ctx)).To(Succeed())
	return adapter
}

func GetAdapterTests() []AdapterTest {
	return []AdapterTest{
		&EtcdAdapterTest{},
		&RedisAdapterTest{},
	}
}

type EtcdAdapterTest struct {
}

func (t *EtcdAdapterTest) Name() string {
	return "Etcd"
}

func (t *EtcdAdapterTest) SetupEnv(envPrefix string) {
	os.Setenv(envPrefix+"_ETCDENDPOINTS", etcdtest.EtcdProxy.ListenAddr())
	os.Setenv(envPrefix+"_ETCDLOCKKEY", "/lockkey"+testhelpers.Rand())
	os.Setenv(envPrefix+"_ETCDADDRKEY", "/addrkey"+testhelpers.Rand())
}

func (t *EtcdAdapterTest) SetLockTTL(envPrefix string, ttlSeconds int) {
	os.Setenv(envPrefix+"_ETCDLOCKTTL", fmt.Sprintf("%d", ttlSeconds))
}

func (t *EtcdAdapterTest) SetUnlockTimeout(envPrefix string, timeout time.Duration) {
	os.Setenv(envPrefix+"_ETCDUNLOCKTIMEOUT", timeout.String())
}

func (t *EtcdAdapterTest) GetAdapter(envPrefix string) Adapter {
	etcdCfg := &etcdadapter.EtcdConfig{}

	err := envconfig.Process(envPrefix, etcdCfg)
	Expect(err).NotTo(HaveOccurred())

	return etcdadapter.NewEtcdAdapter(etcdCfg, Logger)
}

func (t *EtcdAdapterTest) Abort() {
	Expect(etcdtest.EtcdProxy.Close()).To(Succeed())
}

func (t *EtcdAdapterTest) Resume() {
	Expect(etcdtest.EtcdProxy.Start()).To(Succeed())
}

type RedisAdapterTest struct {
}

func (t *RedisAdapterTest) Name() string {
	return "Redis"
}

func (t *RedisAdapterTest) SetupEnv(envPrefix string) {
	os.Setenv(envPrefix+"_REDISURL", "redis://"+redistest.RedisProxy.ListenAddr())
	os.Setenv(envPrefix+"_REDISLOCKKEY", "lockkey"+testhelpers.Rand())
	os.Setenv(envPrefix+"_REDISADDRKEY", "addrkey"+testhelpers.Rand())
}

func (t *RedisAdapterTest) SetLockTTL(envPrefix string, ttlSeconds int) {
	os.Setenv(envPrefix+"_REDISLOCKTTL", fmt.Sprintf("%d", ttlSeconds))
}

func (t *RedisAdapterTest) SetUnlockTimeout(envPrefix string, timeout time.Duration) {
	os.Setenv(envPrefix+"_REDISUNLOCKTIMEOUT", timeout.String())
}

func (t *RedisAdapterTest) GetAdapter(envPrefix string) Adapter {
	redisCfg := &redisadapter.RedisConfig{}

	err := envconfig.Process(envPrefix, redisCfg)
	Expect(err).NotTo(HaveOccurred())

	return redisadapter.NewRedisAdapter(redisCfg, Logger)
}

func (t *RedisAdapterTest) Abort() {
	Expect(redistest.RedisProxy.Close()).To(Succeed())
}

func (t *RedisAdapterTest) Resume() {
	Expect(redistest.RedisProxy.Start()).To(Succeed())
}
