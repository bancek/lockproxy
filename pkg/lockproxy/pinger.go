package lockproxy

import (
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/sirupsen/logrus"
)

type Pinger struct {
	ping                 func(ctx context.Context) error
	retryInitialInterval time.Duration
	retryMaxElapsedTime  time.Duration
	delay                time.Duration
	initialDelay         time.Duration
	logger               *logrus.Entry
}

func NewPinger(
	ping func(ctx context.Context) error,
	retryInitialInterval time.Duration,
	retryMaxElapsedTime time.Duration,
	delay time.Duration,
	initialDelay time.Duration,
	logger *logrus.Entry,
) *Pinger {
	return &Pinger{
		ping:                 ping,
		retryInitialInterval: retryInitialInterval,
		retryMaxElapsedTime:  retryMaxElapsedTime,
		delay:                delay,
		initialDelay:         initialDelay,
		logger:               logger,
	}
}

func (p *Pinger) Ping(ctx context.Context) error {
	p.logger.WithFields(logrus.Fields{
		"maxElapsedTime": p.retryMaxElapsedTime,
	}).Debug("Pinger ping")

	ctx, cancel := context.WithTimeout(ctx, p.retryMaxElapsedTime)
	defer cancel()

	return backoff.RetryNotify(
		func() error {
			return p.ping(ctx)
		},
		backoff.WithContext(GetExponentialBackOff(p.retryInitialInterval, p.retryMaxElapsedTime), ctx),
		func(err error, next time.Duration) {
			p.logger.WithFields(logrus.Fields{
				"nextRetryIn":   next,
				logrus.ErrorKey: err,
			}).Warn("Pinger ping error")
		},
	)
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
