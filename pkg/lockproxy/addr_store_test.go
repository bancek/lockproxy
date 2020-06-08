package lockproxy_test

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"golang.org/x/sync/errgroup"

	. "github.com/bancek/lockproxy/pkg/lockproxy"
)

var _ = Describe("AddrStore", func() {
	var addrKey string

	buildStore := func() *AddrStore {
		s := NewAddrStore(EtcdClient, addrKey, Logger)
		Expect(s.Init(TestCtx)).To(Succeed())
		return s
	}

	BeforeEach(func() {
		addrKey = "/" + Rand()
	})

	Describe("Init", func() {
		It("should initialize the store with empty address", func() {
			s := buildStore()
			Expect(s.Addr()).To(BeEmpty())
		})

		It("should initialize the store with an address", func() {
			s := NewAddrStore(EtcdClient, addrKey, Logger)

			addr := Rand()
			Expect(s.SetAddr(TestCtx, addr)).To(Succeed())

			Expect(s.Init(TestCtx)).To(Succeed())

			Expect(s.Addr()).To(Equal(addr))
		})

		It("should fail if etcd client proxy is closed", func() {
			s := NewAddrStore(EtcdClient, addrKey, Logger)
			EtcdProxy.Close()

			ctx, cancel := context.WithTimeout(TestCtx, 300*time.Millisecond)
			defer cancel()

			Expect(s.Init(ctx)).NotTo(Succeed())
		})
	})

	Describe("Watch", func() {
		It("should watch for changes", func() {
			s1 := buildStore()
			s2 := buildStore()

			watch1Created := make(chan struct{}, 1)
			watch2Created := make(chan struct{}, 1)

			ctx, cancel := context.WithCancel(TestCtx)
			defer cancel()
			g, ctx := errgroup.WithContext(ctx)
			g.Go(func() error {
				return s1.Watch(ctx, func() {
					watch1Created <- struct{}{}
				})
			})
			g.Go(func() error {
				return s2.Watch(ctx, func() {
					watch2Created <- struct{}{}
				})
			})

			Eventually(watch1Created).Should(Receive())
			Eventually(watch2Created).Should(Receive())

			addr := Rand()

			Expect(s2.SetAddr(TestCtx, addr)).To(Succeed())

			Eventually(s1.Addr).Should(Equal(addr))
			Eventually(s2.Addr).Should(Equal(addr))

			cancel()

			Expect(g.Wait()).To(Succeed())
		})
	})
})
