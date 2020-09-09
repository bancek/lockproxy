package redisadapter_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	"github.com/bancek/lockproxy/pkg/lockproxy/redisadapter/redistest"
	"github.com/bancek/lockproxy/pkg/lockproxy/testhelpers"
)

var TestCtx context.Context
var TestCtxTimeoutCancel func()
var Logger *logrus.Entry

func TestRedisAdapter(t *testing.T) {
	if !redistest.RedisInitTesting(t) {
		t.Fatal("Redis not initialized")
	}

	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(10 * time.Second)
	SetDefaultEventuallyPollingInterval(10 * time.Millisecond)

	RunSpecs(t, "RedisAdapter Suite")
}

var _ = BeforeEach(func() {
	TestCtx, TestCtxTimeoutCancel = context.WithTimeout(context.Background(), 10*time.Second)

	redistest.RedisBeforeEach()

	Logger = testhelpers.NewLoggerEntry()
})

var _ = AfterEach(func() {
	redistest.RedisAfterEach()

	TestCtxTimeoutCancel()
})
