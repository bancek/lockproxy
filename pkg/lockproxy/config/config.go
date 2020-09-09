package config

import (
	"time"

	"github.com/kballard/go-shellquote"
)

// Config contains the configuration for the lockproxy
type Config struct {
	// Adapter is the adapter to use ("etcd" or "redis", "etcd" by default).
	// (LOCKPROXY_ADAPTER)
	Adapter string `default:"etcd"`

	// UpstreamAddr is the address of the server started by Cmd.
	// (LOCKPROXY_UPSTREAMADDR)
	UpstreamAddr string `default:"localhost:4080"`

	// UpstreamClientCert is the path of the TLS client certificate.
	// (LOCKPROXY_UPSTREAMCLIENTCERT)
	UpstreamClientCert string

	// UpstreamClientKey is the path of the TLS client certificate key.
	// (LOCKPROXY_UPSTREAMCLIENTKEY)
	UpstreamClientKey string

	// UpstreamClientCA is the path of the TLS server CA.
	// (LOCKPROXY_UPSTREAMCLIENTCA)
	UpstreamClientCA string

	// ProxyListenAddr is the address of the proxy server.
	// (LOCKPROXY_ProxyListenAddr)
	ProxyListenAddr string `default:"localhost:4081"`

	// ProxyRequestAbortTimeout is the duration after which the connection to an
	// upstream is aborted after the current upstream address changes.
	// (LOCKPROXY_PROXYREQUESTABORTTIMEOUT)
	ProxyRequestAbortTimeout time.Duration `default:"10s"`

	// ProxyGrpcMaxCallRecvMsgSize is the maximum message size in bytes the proxy
	// can receive.
	// (LOCKPROXY_PROXYGRPCMAXCALLRECVMSGSIZE)
	ProxyGrpcMaxCallRecvMsgSize int `default:"4194304"`

	// ProxyGrpcMaxCallSendMsgSize is the maximum message size in bytes the proxy
	// can send.
	// (LOCKPROXY_PROXYGRPCMAXCALLSENDMSGSIZE)
	ProxyGrpcMaxCallSendMsgSize int `default:"4194304"`

	// ProxyHealthFollowerInternal controls whether gRPC health checks should be proxied to the
	// internal health server when the current instance is a follower.
	// (LOCKPROXY_PROXYHEALTHFOLLOWERINTERNAL)
	ProxyHealthFollowerInternal bool `default:"true"`

	// HealthListenAddr is the address of the gRPC Health server.
	// (LOCKPROXY_HEALTHLISTENADDR)
	HealthListenAddr string `default:"localhost:4082"`

	// DebugListenAddr is the address of the HTTP debug (pprof) server.
	// (LOCKPROXY_DEBUGLISTENADDR).
	DebugListenAddr string `default:"localhost:4083"`

	// Cmd is the command of the server. Arguments are separated using spaces and
	// can be quoted.
	// (LOCKPROXY_CMD)
	Cmd Cmd

	// CmdShutdownTimeout is the timeout after which the command will be
	// forecefully killed after the proxy is stopped. Command will first receive
	// SIGINT and then SIGKILL after CmdShutdownTimeout.
	// (LOCKPROXY_CMDSHUTDOWNTIMEOUT)
	CmdShutdownTimeout time.Duration `default:"10s"`

	// LogLevel is the log level.
	// (LOCKPROXY_LOGLEVEL)
	LogLevel string `default:"info"`
}

type Cmd []string

func (c *Cmd) Decode(value string) error {
	cmd, err := shellquote.Split(value)
	if err != nil {
		return err
	}
	*c = cmd
	return nil
}
