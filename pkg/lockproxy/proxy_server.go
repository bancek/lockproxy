package lockproxy

import (
	"github.com/mwitkow/grpc-proxy/proxy"
	"google.golang.org/grpc"
)

func NewProxyServer(proxyDirector *ProxyDirector) *grpc.Server {
	return grpc.NewServer(
		grpc.CustomCodec(proxy.Codec()),
		grpc.UnknownServiceHandler(proxy.TransparentHandler(proxyDirector.Director)),
	)
}
