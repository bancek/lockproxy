# lockproxy

Lockproxy is a gRPC proxy. It uses etcd to make sure that only one instance of
proxy starts the upstream service and other instances proxy their requests to
it.

## Running

```sh
export LOCKPROXY_CMD="/path/to/cmd arg1 arg2"
export LOCKPROXY_ETCDLOCKKEY="/applock"
export LOCKPROXY_ETCDADDRKEY="/leaderaddr"
export LOCKPROXY_LOGLEVEL="debug"

UPSTREAMCMD_PORT=1080 \
  LOCKPROXY_UPSTREAMADDR=localhost:1080 \
  LOCKPROXY_PROXYLISTENADDR=localhost:1081 \
  LOCKPROXY_HEALTHLISTENADDR=localhost:1082 \
  LOCKPROXY_DEBUGLISTENADDR=localhost:1083 \
  go run main.go

UPSTREAMCMD_PORT=2080 \
  LOCKPROXY_UPSTREAMADDR=localhost:2080 \
  LOCKPROXY_PROXYLISTENADDR=localhost:2081 \
  LOCKPROXY_HEALTHLISTENADDR=localhost:2082 \
  LOCKPROXY_DEBUGLISTENADDR=localhost:2083 \
  go run main.go

UPSTREAMCMD_PORT=3080 \
  LOCKPROXY_UPSTREAMADDR=localhost:3080 \
  LOCKPROXY_PROXYLISTENADDR=localhost:3081 \
  LOCKPROXY_HEALTHLISTENADDR=localhost:3082 \
  LOCKPROXY_DEBUGLISTENADDR=localhost:3083 \
  go run main.go

grpc-health-probe -addr 127.0.0.1:1081
grpc-health-probe -addr 127.0.0.1:2081
grpc-health-probe -addr 127.0.0.1:3081
```

## Testing

```sh
etcd

export ETCD_ENDPOINT="localhost:2379"

go test ./...
```

### Coverage

```sh
ginkgo --cover ./lockproxy && go tool cover -html=./lockproxy/lockproxy.coverprofile -o ./lockproxy/lockproxy.coverprofile.html
```

## Debug

Get a list of goroutines:

```sh
curl localhost:4083/debug/pprof/goroutine?debug=2
```
