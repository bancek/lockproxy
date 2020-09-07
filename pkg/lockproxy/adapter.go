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
