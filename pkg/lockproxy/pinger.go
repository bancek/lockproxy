package lockproxy

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"go.etcd.io/etcd/clientv3"
	"golang.org/x/xerrors"
)

type Pinger struct {
	etcdClient   *clientv3.Client
	pingKey      string
	pingTimeout  time.Duration
	delay        time.Duration
	initialDelay time.Duration
	logger       *logrus.Entry
}

func NewPinger(
	etcdClient *clientv3.Client,
	pingKey string,
	pingTimeout time.Duration,
	delay time.Duration,
	initialDelay time.Duration,
	logger *logrus.Entry,
) *Pinger {
	return &Pinger{
		etcdClient:   etcdClient,
		pingKey:      pingKey,
		pingTimeout:  pingTimeout,
		delay:        delay,
		initialDelay: initialDelay,
		logger:       logger,
	}
}

func (p *Pinger) Ping(ctx context.Context) error {
	p.logger.WithFields(logrus.Fields{
		"pingKey":     p.pingKey,
		"pingTimeout": p.pingTimeout,
	}).Debug("Pinger ping")

	getCtx, cancel := context.WithTimeout(ctx, p.pingTimeout)
	defer cancel()

	_, err := p.etcdClient.Get(getCtx, p.pingKey)
	if err != nil {
		return xerrors.Errorf("ping failed: %s: %w", p.pingKey, err)
	}

	return nil
}

func (p *Pinger) Start(ctx context.Context) error {
	delay := p.delay

	if p.initialDelay >= 0 {
		delay = p.initialDelay
	}

	for {
		timer := time.NewTimer(delay)

		select {
		case <-timer.C:
			if err := p.Ping(ctx); err != nil {
				return err
			}

			delay = p.delay

		case <-ctx.Done():
			timer.Stop()

			return nil
		}
	}
}
