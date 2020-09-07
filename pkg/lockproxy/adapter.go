package lockproxy

import "context"

type Locker interface {
	Start(ctx context.Context) error
}
