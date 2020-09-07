package config

import (
	"time"

	"github.com/kballard/go-shellquote"
)

// Config contains the configuration for the lockproxy
type Config struct {
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

	// ProxyRequestAbortTimeout is the duration after which the connection to
	// an upstream is aborted after the current upstream address changes.
	// (LOCKPROXY_PROXYREQUESTABORTTIMEOUT)
	ProxyRequestAbortTimeout time.Duration `default:"10s"`

	// ProxyGrpcMaxCallRecvMsgSize is the maximum message size in bytes the proxy can receive.
	// (LOCKPROXY_PROXYGRPCMAXCALLRECVMSGSIZE)
	ProxyGrpcMaxCallRecvMsgSize int `default:"4194304"`

	// ProxyGrpcMaxCallSendMsgSize is the maximum message size in bytes the proxy can send.
	// (LOCKPROXY_PROXYGRPCMAXCALLSENDMSGSIZE)
	ProxyGrpcMaxCallSendMsgSize int `default:"4194304"`

	// HealthListenAddr is the address of the gRPC Health server. It should
	// only be used internally. Health probes should be directed to ListenAddr.
	// (LOCKPROXY_HEALTHLISTENADDR)
	HealthListenAddr string `default:"localhost:4082"`

	// DebugListenAddr is the address of the HTTP debug (pprof) server
	// (LOCKPROXY_DEBUGLISTENADDR).
	DebugListenAddr string `default:"localhost:4083"`

	// EtcdEndpoints is a comma separated list etcd server endpoints.
	// (LOCKPROXY_ETCDENDPOINTS)
	EtcdEndpoints []string `default:"localhost:2379"`

	// EtcdDialTimeout is the timeout for failing to establish an etcd connection.
	// (LOCKPROXY_ETCDDIALTIMEOUT)
	EtcdDialTimeout time.Duration `default:"10s"`

	// EtcdDialKeepAliveTime is the time after which client pings the server to
	// see if the transport is alive.
	// (LOCKPROXY_ETCDDIALKEEPALIVETIME)
	EtcdDialKeepAliveTime time.Duration `default:"10s"`

	// DialKeepAliveTimeout is the time that the client waits for a response for
	// the keep-alive probe. If the response is not received in this time, the
	// connection is closed.
	// (LOCKPROXY_ETCDDIALKEEPALIVETIMEOUT)
	EtcdDialKeepAliveTimeout time.Duration `default:"10s"`

	// EtcdUsername is an etcd user name for authentication.
	// (LOCKPROXY_ETCDUSERNAME)
	EtcdUsername string

	// EtcdPassword is an etcd password for authentication.
	// (LOCKPROXY_ETCDPASSWORD)
	EtcdPassword string

	// EtcdLockTTL is the duration of the etcd lock (in seconds). The lock will
	// be refreshed every EtcdLockTTL seconds.
	// (LOCKPROXY_ETCDLOCKTTL)
	EtcdLockTTL int `default:"10"`

	// EtcdUnlockTimeout is the max duration for waiting etcd to unlock after the
	// Cmd is stopped.
	// (LOCKPROXY_ETCDUNLOCKTIMEOUT)
	EtcdUnlockTimeout time.Duration `default:"10s"`

	// EtcdLockKey is the etcd key used for etcd lock.
	// (LOCKPROXY_ETCDLOCKKEY)
	EtcdLockKey string `required:"true"`

	// EtcdAddrKey is the etcd key used to store the address of the current
	// leader.
	// (LOCKPROXY_ETCDADDRKEY)
	EtcdAddrKey string `required:"true"`

	// EtcdPingTimeout is the timeout after which the ping is deemed failed and the
	// process will exit.
	// (LOCKPROXY_ETCDPINGTIMEOUT)
	EtcdPingTimeout time.Duration `default:"10s"`

	// EtcdPingDelay is the delay between pings.
	// (LOCKPROXY_ETCDPINGDELAY)
	EtcdPingDelay time.Duration `default:"10s"`

	// EtcdPingInitialDelay is the delay before the first ping.
	// (LOCKPROXY_ETCDPINGINITIALDELAY)
	EtcdPingInitialDelay time.Duration `default:"10s"`

	// Cmd is the command of the server. Arguments are separated using spaces and
	// can be quoted.
	// (LOCKPROXY_CMD)
	Cmd Cmd `required:"true"`

	// CmdShutdownTimeout is the timeout after which the command will be
	// forecefully killed after the proxy is stopped. Command will first
	// receive SIGINT and then SIGKILL after CmdShutdownTimeout.
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
