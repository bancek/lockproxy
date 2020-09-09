package lockproxy_test

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc"
	"google.golang.org/grpc/examples/helloworld/helloworld"
	"google.golang.org/grpc/health/grpc_health_v1"

	. "github.com/bancek/lockproxy/pkg/lockproxy"
	"github.com/bancek/lockproxy/pkg/lockproxy/config"
	"github.com/bancek/lockproxy/pkg/lockproxy/etcdadapter"
	"github.com/bancek/lockproxy/pkg/lockproxy/redisadapter"
	"github.com/bancek/lockproxy/pkg/lockproxy/redisadapter/redistest"
	"github.com/bancek/lockproxy/pkg/lockproxy/testhelpers"
)

var _ = Describe("LockProxy", func() {
	buildDummyCmd := func(tmpDir string) string {
		dummyCmdPath := filepath.Join(tmpDir, "dummycmd")

		cmd := exec.Command("go", "build", "-o", dummyCmdPath, "./dummycmd/dummycmd.go")
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).To(Succeed())

		return dummyCmdPath
	}

	generateTests := func(
		setupEnv func(envPrefix string),
		getAdapter func(envPrefix string) Adapter,
	) {
		It("should run the proxy", func() {
			tmpDir, err := ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			dummyCmdPath := buildDummyCmd(tmpDir)

			cfg := &config.Config{}

			envPrefix := strings.ToUpper("LOCKPROXYTEST" + testhelpers.Rand())

			os.Setenv(envPrefix+"_CMD", dummyCmdPath+" -addr 127.0.0.1:4080")
			setupEnv(envPrefix)

			err = envconfig.Process(envPrefix, cfg)
			Expect(err).NotTo(HaveOccurred())

			adapter := getAdapter(envPrefix)

			proxy := NewLockProxy(cfg, adapter, Logger)

			startErr := make(chan error, 1)

			func() {
				ctx, cancel := context.WithCancel(TestCtx)
				defer cancel()

				err = proxy.Init(ctx)
				Expect(err).NotTo(HaveOccurred())
				defer proxy.Close()

				go func() {
					startErr <- proxy.Start()
				}()

				conn, err := grpc.DialContext(ctx, "127.0.0.1:4081", grpc.WithInsecure())
				Expect(err).NotTo(HaveOccurred())
				defer conn.Close()

				healthClient := grpc_health_v1.NewHealthClient(conn)

				Eventually(func() error {
					_, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
					return err
				}, 10*time.Second, 100*time.Millisecond).Should(Succeed())

				resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp.Status).To(Equal(grpc_health_v1.HealthCheckResponse_SERVING))

				name := testhelpers.Rand()

				greeterClient := helloworld.NewGreeterClient(conn)

				Eventually(func() error {
					_, err := greeterClient.SayHello(ctx, &helloworld.HelloRequest{
						Name: name,
					})
					return err
				}, 10*time.Second, 100*time.Millisecond).Should(Succeed())
				resp1, err := greeterClient.SayHello(ctx, &helloworld.HelloRequest{
					Name: name,
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(resp1.Message).To(Equal("Hello " + name))
			}()

			Eventually(startErr).Should(Receive())
		})
	}

	Describe("EtcdAdapter", func() {
		generateTests(
			func(envPrefix string) {
				os.Setenv(envPrefix+"_ETCDLOCKKEY", "/lockkey"+testhelpers.Rand())
				os.Setenv(envPrefix+"_ETCDADDRKEY", "/addrkey"+testhelpers.Rand())
			},
			func(envPrefix string) Adapter {
				etcdCfg := &etcdadapter.EtcdConfig{}

				err := envconfig.Process(envPrefix, etcdCfg)
				Expect(err).NotTo(HaveOccurred())

				return etcdadapter.NewEtcdAdapter(etcdCfg, Logger)
			},
		)
	})

	Describe("RedisAdapter", func() {
		generateTests(
			func(envPrefix string) {
				os.Setenv(envPrefix+"_REDISURL", "redis://"+redistest.RedisAddr)
				os.Setenv(envPrefix+"_REDISLOCKKEY", "lockkey"+testhelpers.Rand())
				os.Setenv(envPrefix+"_REDISADDRKEY", "addrkey"+testhelpers.Rand())
			},
			func(envPrefix string) Adapter {
				redisCfg := &redisadapter.RedisConfig{}

				err := envconfig.Process(envPrefix, redisCfg)
				Expect(err).NotTo(HaveOccurred())

				return redisadapter.NewRedisAdapter(redisCfg, Logger)
			},
		)
	})
})
