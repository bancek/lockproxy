package lockproxy_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/bancek/lockproxy/pkg/lockproxy"
	"github.com/bancek/lockproxy/pkg/lockproxy/testhelpers"
)

var _ = Describe("RemoteAddrStore", func() {
	generateTests := func(adapterTest AdapterTest) {
		Describe(adapterTest.Name(), func() {
			BeforeEach(func() {
				adapterTest.SetupEnv(EnvPrefix)
			})

			buildStore := func() (RemoteAddrStore, func(canStartFail bool)) {
				ctx, cancel := context.WithCancel(TestCtx)
				adapter := BuildAdapter(ctx, adapterTest)

				localAddrStore := NewMemoryAddrStore(Logger)
				s, err := adapter.GetRemoteAddrStore(localAddrStore)
				Expect(err).NotTo(HaveOccurred())

				onStartedCh := make(chan struct{})
				startErrCh := make(chan error)

				go func() {
					startErrCh <- s.Start(ctx, func() {
						onStartedCh <- struct{}{}
					})
				}()

				Eventually(onStartedCh).Should(Receive())

				return s, func(canStartFail bool) {
					cancel()

					var err error

					Eventually(startErrCh).Should(Receive(&err))

					if !canStartFail {
						Expect(err).NotTo(HaveOccurred())
					}
				}
			}

			It("should start with an empty addr", func() {
				s, done := buildStore()
				defer done(false)

				Expect(s.Addr(TestCtx)).To(BeEmpty())
			})

			It("should set addr", func() {
				s1, done1 := buildStore()
				defer done1(false)
				s2, done2 := buildStore()
				defer done2(false)

				addr := testhelpers.Rand()

				Expect(s1.SetAddr(TestCtx, addr)).To(Succeed())

				Eventually(func() string {
					return s1.Addr(TestCtx)
				}).Should(Equal(addr))
				Eventually(func() string {
					return s2.Addr(TestCtx)
				}).Should(Equal(addr))
			})

			It("should work if network temporarily breaks", func() {
				s1, done1 := buildStore()
				defer done1(false)
				s2, done2 := buildStore()
				defer done2(false)

				addr1 := testhelpers.Rand()

				Expect(s1.SetAddr(TestCtx, addr1)).To(Succeed())

				Eventually(func() string {
					return s1.Addr(TestCtx)
				}).Should(Equal(addr1))
				Eventually(func() string {
					return s2.Addr(TestCtx)
				}).Should(Equal(addr1))

				adapterTest.Abort()

				time.Sleep(200 * time.Millisecond)

				adapterTest.Resume()

				addr2 := testhelpers.Rand()

				Expect(s1.SetAddr(TestCtx, addr2)).To(Succeed())

				Eventually(func() string {
					return s1.Addr(TestCtx)
				}).Should(Equal(addr2))
				Eventually(func() string {
					return s2.Addr(TestCtx)
				}).Should(Equal(addr2))
			})

			It("should stop if network breaks and ctx is closed", func() {
				s1, done1 := buildStore()
				defer done1(true)
				s2, done2 := buildStore()
				defer done2(true)

				addr1 := testhelpers.Rand()

				Expect(s1.SetAddr(TestCtx, addr1)).To(Succeed())

				Eventually(func() string {
					return s1.Addr(TestCtx)
				}).Should(Equal(addr1))
				Eventually(func() string {
					return s2.Addr(TestCtx)
				}).Should(Equal(addr1))

				adapterTest.Abort()

				time.Sleep(200 * time.Millisecond)
			})

			It("should retry SetAddr if network temporarily breaks", func() {
				s1, done1 := buildStore()
				defer done1(false)
				s2, done2 := buildStore()
				defer done2(false)

				addr1 := testhelpers.Rand()

				Expect(s1.SetAddr(TestCtx, addr1)).To(Succeed())

				Eventually(func() string {
					return s1.Addr(TestCtx)
				}).Should(Equal(addr1))
				Eventually(func() string {
					return s2.Addr(TestCtx)
				}).Should(Equal(addr1))

				adapterTest.Abort()

				time.Sleep(100 * time.Millisecond)

				go func() {
					time.Sleep(100 * time.Millisecond)

					adapterTest.Resume()
				}()

				addr2 := testhelpers.Rand()

				Expect(s1.SetAddr(TestCtx, addr2)).To(Succeed())

				Eventually(func() string {
					return s1.Addr(TestCtx)
				}).Should(Equal(addr2))
				Eventually(func() string {
					return s2.Addr(TestCtx)
				}).Should(Equal(addr2))
			})
		})
	}

	for _, adapterTest := range GetAdapterTests() {
		generateTests(adapterTest)
	}
})
