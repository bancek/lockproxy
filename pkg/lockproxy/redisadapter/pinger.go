package redisadapter

import (
	"context"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

type Pinger struct {
	redisPool    *redis.Pool
	pingKey      string
	pingTimeout  time.Duration
	delay        time.Duration
	initialDelay time.Duration
	logger       *logrus.Entry
}

func NewPinger(
	redisPool *redis.Pool,
	pingKey string,
	pingTimeout time.Duration,
	delay time.Duration,
	initialDelay time.Duration,
	logger *logrus.Entry,
) *Pinger {
	return &Pinger{
		redisPool:    redisPool,
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

	conn, err := p.redisPool.GetContext(getCtx)
	if err != nil {
		return xerrors.Errorf("failed to get redis conn: %w", err)
	}
	defer conn.Close()

	_, err = conn.Do("PING")
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
