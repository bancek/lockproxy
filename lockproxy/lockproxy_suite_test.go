package lockproxy_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/bancek/tcpproxy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"

	. "github.com/bancek/lockproxy/lockproxy"
)

var TestCtx context.Context
var TestCtxTimeoutCancel func()
var EtcdEndpoint string
var EtcdProxy *tcpproxy.Proxy
var EtcdClient *clientv3.Client
var Logger *logrus.Entry

func TestLockproxy(t *testing.T) {
	EtcdEndpoint = os.Getenv("ETCD_ENDPOINT")
	if EtcdEndpoint == "" {
		t.Fatal("Missing ETCD_ENDPOINT env variable")
	}

	RegisterFailHandler(Fail)

	SetDefaultEventuallyTimeout(10 * time.Second)
	SetDefaultEventuallyPollingInterval(10 * time.Millisecond)

	RunSpecs(t, "Lockproxy Suite")
}

var _ = BeforeEach(func() {
	var err error

	TestCtx, TestCtxTimeoutCancel = context.WithTimeout(context.Background(), 10*time.Second)

	EtcdProxy, err = tcpproxy.NewUnusedAddr("127.0.0.1", EtcdEndpoint)
	Expect(err).NotTo(HaveOccurred())
	Expect(EtcdProxy.Start()).To(Succeed())

	EtcdClient, err = NewEtcdClient(TestCtx, &Config{
		EtcdEndpoints:            []string{EtcdProxy.ListenAddr()},
		EtcdDialTimeout:          10 * time.Second,
		EtcdDialKeepAliveTime:    2 * time.Second,
		EtcdDialKeepAliveTimeout: 2 * time.Second,
	})
	Expect(err).NotTo(HaveOccurred())

	Logger = NewLoggerEntry()
})

var _ = AfterEach(func() {
	EtcdClient.Close()
	EtcdClient = nil

	EtcdProxy.Close()
	EtcdProxy = nil

	TestCtxTimeoutCancel()
})
