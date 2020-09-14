package lockproxy_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/bancek/lockproxy/pkg/lockproxy/etcdadapter/etcdtest"
	"github.com/bancek/lockproxy/pkg/lockproxy/redisadapter/redistest"
	"github.com/bancek/lockproxy/pkg/lockproxy/testhelpers"
)

var TestCtx context.Context
var TestCtxTimeoutCancel func()
var Logger *logrus.Entry
var EnvPrefix string

func TestLockproxy(t *testing.T) {
	if !etcdtest.EtcdInitTesting(t) {
		t.Fatal("Etcd not initialized")
	}
	if !redistest.RedisInitTesting(t) {
		t.Fatal("Redis not initialized")
	}

	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(10 * time.Second)
	SetDefaultEventuallyPollingInterval(10 * time.Millisecond)

	RunSpecs(t, "Lockproxy Suite")
}

var _ = BeforeEach(func() {
	TestCtx, TestCtxTimeoutCancel = context.WithTimeout(context.Background(), 10*time.Second)

	etcdtest.EtcdBeforeEach(TestCtx)
	redistest.RedisBeforeEach()

	Logger = testhelpers.NewLoggerEntry()

	EnvPrefix = testhelpers.GetEnvPrefix()
})

var _ = AfterEach(func() {
	etcdtest.EtcdAfterEach()
	redistest.RedisAfterEach()

	TestCtxTimeoutCancel()
})
