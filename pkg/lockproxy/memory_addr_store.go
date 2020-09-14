package lockproxy

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

// MemoryAddrStore stores the current proxy address locally.
type MemoryAddrStore struct {
	logger *logrus.Entry

	addr  string
	mutex sync.RWMutex
}

func NewMemoryAddrStore(
	logger *logrus.Entry,
) *MemoryAddrStore {
	return &MemoryAddrStore{
		logger: logger,

		addr: "",
	}
}

func (s *MemoryAddrStore) Addr(ctx context.Context) string {
	s.mutex.RLock()
	addr := s.addr
	s.mutex.RUnlock()
	return addr
}

func (s *MemoryAddrStore) SetAddr(ctx context.Context, addr string) {
	s.logger.WithFields(logrus.Fields{
		"addr": addr,
	}).Debug("MemoryAddrStore setting addr")

	s.mutex.Lock()
	s.addr = addr
	s.mutex.Unlock()
}

var _ LocalAddrStore = (*MemoryAddrStore)(nil)
