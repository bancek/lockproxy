package etcdadapter

import (
	"context"

	"github.com/bancek/lockproxy/pkg/lockproxy/config"
	"go.etcd.io/etcd/clientv3"
	"google.golang.org/grpc"
)

func NewEtcdClient(ctx context.Context, config *config.Config) (*clientv3.Client, error) {
	return clientv3.New(clientv3.Config{
		Endpoints:            config.EtcdEndpoints,
		DialTimeout:          config.EtcdDialTimeout,
		DialKeepAliveTime:    config.EtcdDialKeepAliveTime,
		DialKeepAliveTimeout: config.EtcdDialKeepAliveTimeout,
		Username:             config.EtcdUsername,
		Password:             config.EtcdPassword,
		DialOptions:          []grpc.DialOption{grpc.WithBlock()},
		Context:              ctx,
	})
}
