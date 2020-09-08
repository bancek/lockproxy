package etcdadapter

import "time"

type EtcdConfig struct {
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
}
