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
)

var _ = Describe("LockProxy", func() {
	It("should run the proxy", func() {
		tmpDir, err := ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())

		dummyCmdPath := filepath.Join(tmpDir, "dummycmd")

		cmd := exec.Command("go", "build", "-o", dummyCmdPath, "./dummycmd/dummycmd.go")
		cmd.Stdout = GinkgoWriter
		cmd.Stderr = GinkgoWriter
		Expect(cmd.Run()).To(Succeed())

		config := Config{}

		prefix := strings.ToUpper("LOCKPROXYTEST" + Rand())

		os.Setenv(prefix+"_ETCDLOCKKEY", "/lockkey"+Rand())
		os.Setenv(prefix+"_ETCDADDRKEY", "/addrkey"+Rand())
		os.Setenv(prefix+"_CMD", dummyCmdPath+" -addr 127.0.0.1:4080")

		err = envconfig.Process(prefix, &config)
		Expect(err).NotTo(HaveOccurred())

		proxy := NewLockProxy(&config, Logger)

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
			resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.Status).To(Equal(grpc_health_v1.HealthCheckResponse_SERVING))

			name := Rand()

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
})
