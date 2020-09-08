package etcdtest

import (
	"context"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"go.etcd.io/etcd/clientv3"

	"github.com/bancek/lockproxy/pkg/lockproxy/etcdadapter"
	"github.com/bancek/tcpproxy"
)

var EtcdEndpoint string
var EtcdProxy *tcpproxy.Proxy
var EtcdClient *clientv3.Client

func EtcdInitTesting(t *testing.T) bool {
	EtcdEndpoint = os.Getenv("ETCD_ENDPOINT")
	if EtcdEndpoint == "" {
		t.Skip("Missing ETCD_ENDPOINT env variable")
		return false
	}

	return true
}

func EtcdBeforeEach(ctx context.Context) {
	var err error
	EtcdProxy, err = tcpproxy.NewUnusedAddr("127.0.0.1", EtcdEndpoint)
	Expect(err).NotTo(HaveOccurred())
	Expect(EtcdProxy.Start()).To(Succeed())

	EtcdClient, err = etcdadapter.NewEtcdClient(ctx, &etcdadapter.EtcdConfig{
		EtcdEndpoints:            []string{EtcdProxy.ListenAddr()},
		EtcdDialTimeout:          10 * time.Second,
		EtcdDialKeepAliveTime:    2 * time.Second,
		EtcdDialKeepAliveTimeout: 2 * time.Second,
	})
	Expect(err).NotTo(HaveOccurred())
}

func EtcdAfterEach() {
	EtcdClient.Close()
	EtcdClient = nil

	EtcdProxy.Close()
	EtcdProxy = nil
}
