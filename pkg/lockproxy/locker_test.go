package lockproxy_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/xerrors"

	. "github.com/bancek/lockproxy/pkg/lockproxy"
)

var _ = Describe("Locker", func() {
	generateTests := func(adapterTest AdapterTest) {
		Describe(adapterTest.Name(), func() {
			BeforeEach(func() {
				adapterTest.SetupEnv(EnvPrefix)
			})

			buildLocker := func(onLocked func(ctx context.Context) error) (l Locker, ctx context.Context, done func()) {
				ctx, cancel := context.WithCancel(TestCtx)
				adapter := BuildAdapter(ctx, adapterTest)

				l, err := adapter.GetLocker(onLocked)
				Expect(err).NotTo(HaveOccurred())

				return l, ctx, cancel
			}

			It("should obtain a lock", func() {
				locked := false
				l, ctx, done := buildLocker(func(ctx context.Context) error {
					locked = true
					return nil
				})
				defer done()
				Expect(l.Start(ctx)).To(Succeed())
				Expect(locked).To(BeTrue())
			})

			It("should work if network temporarily breaks", func() {
				adapterTest.SetLockTTL(EnvPrefix, 1)

				locked := false
				l, ctx, done := buildLocker(func(ctx context.Context) error {
					locked = true

					adapterTest.Abort()

					time.Sleep(1500 * time.Millisecond)

					adapterTest.Resume()

					return nil
				})
				defer done()
				Expect(l.Start(ctx)).To(Succeed())
				Expect(locked).To(BeTrue())
			})

			It("should fail to extend and stop if network breaks and ctx is closed", func() {
				adapterTest.SetLockTTL(EnvPrefix, 1)
				adapterTest.SetUnlockTimeout(EnvPrefix, 2*time.Second)

				locked := false
				l, ctx, done := buildLocker(func(ctx context.Context) error {
					locked = true

					adapterTest.Abort()

					time.Sleep(1500 * time.Millisecond)

					return ctx.Err()
				})
				defer done()
				err := l.Start(ctx)
				Expect(err).To(HaveOccurred())
				Expect(locked).To(BeTrue())
			})

			It("should stop if network breaks and ctx is closed", func() {
				adapterTest.SetLockTTL(EnvPrefix, 1)
				adapterTest.SetUnlockTimeout(EnvPrefix, 2*time.Second)

				locked := false
				l, ctx, done := buildLocker(func(ctx context.Context) error {
					locked = true

					adapterTest.Abort()

					return nil
				})
				defer done()
				err := l.Start(ctx)
				Expect(err).To(HaveOccurred())
				Expect(locked).To(BeTrue())
			})

			It("should stop if onLocked fails", func() {
				adapterTest.SetLockTTL(EnvPrefix, 1)
				adapterTest.SetUnlockTimeout(EnvPrefix, 2*time.Second)

				locked := false
				l, ctx, done := buildLocker(func(ctx context.Context) error {
					locked = true

					return xerrors.Errorf("custom error")
				})
				defer done()
				err := l.Start(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("custom error"))
				Expect(locked).To(BeTrue())
			})

			It("should fail to obtain a lock if already locked", func() {
				locked1 := make(chan struct{}, 1)
				l1, ctx1, done1 := buildLocker(func(ctx context.Context) error {
					locked1 <- struct{}{}
					<-ctx.Done()
					return nil
				})
				defer done1()

				l1StartErrCh := make(chan error)
				go func() {
					l1StartErrCh <- l1.Start(ctx1)
				}()

				Eventually(locked1).Should(Receive())

				locked2 := make(chan struct{}, 1)
				l2, ctx2, done2 := buildLocker(func(ctx context.Context) error {
					locked2 <- struct{}{}
					return nil
				})
				defer done2()

				l2StartErrCh := make(chan error)

				go func() {
					l2StartErrCh <- l2.Start(ctx2)
				}()

				time.Sleep(1200 * time.Millisecond)

				Expect(locked2).NotTo(Receive())

				done1()
				done2()

				var l1StartErr error
				Eventually(l1StartErrCh).Should(Receive(&l1StartErr))
				var l2StartErr error
				Eventually(l2StartErrCh).Should(Receive(&l2StartErr))
			})

			It("should fail to obtain a lock and stop if already locked and network breaks", func() {
				adapterTest.SetLockTTL(EnvPrefix, 1)
				adapterTest.SetUnlockTimeout(EnvPrefix, 1*time.Second)

				locked1 := make(chan struct{}, 1)
				l1, ctx1, done1 := buildLocker(func(ctx context.Context) error {
					locked1 <- struct{}{}
					<-ctx.Done()
					return nil
				})
				defer done1()

				l1StartErrCh := make(chan error)
				go func() {
					l1StartErrCh <- l1.Start(ctx1)
				}()

				Eventually(locked1).Should(Receive())

				locked2 := make(chan struct{}, 1)
				l2, ctx2, done2 := buildLocker(func(ctx context.Context) error {
					locked2 <- struct{}{}
					return nil
				})
				defer done2()

				l2StartErrCh := make(chan error)

				go func() {
					l2StartErrCh <- l2.Start(ctx2)
				}()

				time.Sleep(1200 * time.Millisecond)

				Expect(locked2).NotTo(Receive())

				adapterTest.Abort()

				time.Sleep(1200 * time.Millisecond)

				done1()
				done2()

				var l1StartErr error
				Eventually(l1StartErrCh).Should(Receive(&l1StartErr))
				var l2StartErr error
				Eventually(l2StartErrCh).Should(Receive(&l2StartErr))
			})
		})
	}

	for _, adapterTest := range GetAdapterTests() {
		generateTests(adapterTest)
	}
})
