package lockproxy_test

import (
	"context"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/xerrors"

	. "github.com/bancek/lockproxy/pkg/lockproxy"
)

var _ = Describe("Pinger", func() {
	var pingCalls int64
	var ping func(ctx context.Context) error
	var retryInitialInterval time.Duration
	var retryMaxElapsedTime time.Duration
	var delay time.Duration
	var initialDelay time.Duration

	pingCalled := func() {
		atomic.AddInt64(&pingCalls, 1)
	}

	getPingCalls := func() int64 {
		return atomic.LoadInt64(&pingCalls)
	}

	BeforeEach(func() {
		pingCalls = 0
		ping = func(ctx context.Context) error {
			pingCalled()
			return nil
		}

		retryInitialInterval = 20 * time.Millisecond
		retryMaxElapsedTime = 200 * time.Millisecond
		delay = 100 * time.Millisecond
		initialDelay = 10 * time.Millisecond
	})

	buildPinger := func() func() error {
		ctx, cancel := context.WithCancel(TestCtx)

		p := NewPinger(
			ping,
			retryInitialInterval,
			retryMaxElapsedTime,
			delay,
			initialDelay,
			Logger,
		)

		startErrCh := make(chan error)

		go func() {
			startErrCh <- p.Start(ctx)
		}()

		return func() error {
			cancel()

			var err error

			Eventually(startErrCh).Should(Receive(&err))

			return err
		}
	}

	It("should ping", func() {
		done := buildPinger()

		Eventually(getPingCalls).Should(BeNumerically(">=", 1))
		Expect(done()).To(Succeed())
	})

	It("should retry", func() {
		ping = func(ctx context.Context) error {
			pingCalled()
			if getPingCalls() > 2 {
				return nil
			}
			return xerrors.Errorf("custom ping error")
		}
		done := buildPinger()

		Eventually(getPingCalls).Should(BeNumerically(">=", 5))
		Expect(done()).To(Succeed())
	})

	It("should fail after max elapsed time", func() {
		pingErr := xerrors.Errorf("custom ping error")
		ping = func(ctx context.Context) error {
			pingCalled()
			return pingErr
		}
		done := buildPinger()

		time.Sleep(retryMaxElapsedTime + 50*time.Millisecond)

		err := done()
		Expect(xerrors.Is(err, pingErr)).To(BeTrue())
	})

	It("should cancel ctx after max elapsed time", func() {
		pingCtxDoneCh := make(chan bool, 1)
		ping = func(ctx context.Context) error {
			time.Sleep(retryMaxElapsedTime + 50*time.Millisecond)

			select {
			case <-ctx.Done():
				pingCtxDoneCh <- true
			default:
				pingCtxDoneCh <- false
			}

			return nil
		}
		done := buildPinger()
		defer func() {
			_ = done()
		}()

		var pingCtxDone bool
		Eventually(pingCtxDoneCh).Should(Receive(&pingCtxDone))
		Expect(pingCtxDone).To(BeTrue())
	})
})
