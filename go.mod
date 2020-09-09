module github.com/bancek/lockproxy

go 1.12

require (
	github.com/bancek/tcpproxy v0.1.0
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/pkg v0.0.0-20180928190104-399ea9e2e55f // indirect
	github.com/go-redsync/redsync/v3 v3.0.0
	github.com/gogo/protobuf v1.3.1 // indirect
	github.com/gomodule/redigo v2.0.0+incompatible
	github.com/google/uuid v1.1.1
	github.com/hashicorp/go-multierror v1.1.0
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/koofr/go-ttl-cache v0.0.0-20190517172119-42f05463ecd2
	github.com/mwitkow/grpc-proxy v0.0.0-20181017164139-0f1106ef9c76
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.6.1
	go.etcd.io/etcd v0.0.0-20200520232829-54ba9589114f
	go.uber.org/zap v1.15.0 // indirect
	golang.org/x/sync v0.0.0-20190423024810-112230192c58
	golang.org/x/xerrors v0.0.0-20191204190536-9bdfabe68543
	google.golang.org/grpc v1.30.0-dev.1
	google.golang.org/grpc/examples v0.0.0-20200605192255-479df5ea818c
)

replace github.com/coreos/go-systemd => github.com/coreos/go-systemd/v22 v22.0.0
