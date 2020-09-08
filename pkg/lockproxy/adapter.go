package lockproxy

import "context"

type Locker interface {
	Start(ctx context.Context) error
}

type AddrStore interface {
	Init(ctx context.Context) error
	Addr() string
	SetAddr(ctx context.Context, addr string) error
	Watch(ctx context.Context, onCreated func()) error
}

type Pinger interface {
	Start(ctx context.Context) error
}

type Adapter interface {
	Init(ctx context.Context) error
	GetLocker(onLocked func(ctx context.Context) error) (Locker, error)
	GetAddrStore() (AddrStore, error)
	GetPinger() (Pinger, error)
	Close() error
}
