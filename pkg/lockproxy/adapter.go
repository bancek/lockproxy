package lockproxy

import "context"

type Locker interface {
	Start(ctx context.Context) error
}

type LocalAddrStore interface {
	Addr(ctx context.Context) string
	SetAddr(ctx context.Context, addr string)
}

type RemoteAddrStore interface {
	Addr(ctx context.Context) string
	Refresh(ctx context.Context) (addr string, err error)
	Start(ctx context.Context, onInit func()) error
	SetAddr(ctx context.Context, addr string) error
}

type Pinger interface {
	Start(ctx context.Context) error
}

type Adapter interface {
	Init(ctx context.Context) error
	GetLocker(onLocked func(ctx context.Context) error) (Locker, error)
	GetRemoteAddrStore(addrStore LocalAddrStore) (RemoteAddrStore, error)
	GetPinger() (Pinger, error)
	Close() error
}
