# lockproxy

Lockproxy is a gRPC proxy. It uses etcd or Redis to make sure that only one
instance of proxy starts the upstream service and other instances proxy their
requests to it.

## Running

```sh
# etcd
etcd

# redis
docker run --rm -p 6381:6379 redis
# redis proxy for testing network errors
socat tcp-listen:6382,reuseaddr,fork tcp:127.0.0.1:6381

go build ./pkg/lockproxy/dummycmd
go build ./pkg/lockproxy/dummyclient
go build main.go

export LOCKPROXY_LOGLEVEL="debug"

# etcd
export LOCKPROXY_ADAPTER="etcd"
export LOCKPROXY_ETCDLOCKKEY="/applock"
export LOCKPROXY_ETCDADDRKEY="/leaderaddr"

# redis
export LOCKPROXY_ADAPTER="redis"
export LOCKPROXY_REDISURL="redis://localhost:6382"
export LOCKPROXY_REDISLOCKKEY="applock"
export LOCKPROXY_REDISADDRKEY="leaderaddr"

LOCKPROXY_UPSTREAMADDR=localhost:1080 \
  LOCKPROXY_PROXYLISTENADDR=localhost:1081 \
  LOCKPROXY_HEALTHLISTENADDR=localhost:1082 \
  LOCKPROXY_DEBUGLISTENADDR=localhost:1083 \
  ./main ./dummycmd -addr localhost:1080

LOCKPROXY_UPSTREAMADDR=localhost:2080 \
  LOCKPROXY_PROXYLISTENADDR=localhost:2081 \
  LOCKPROXY_HEALTHLISTENADDR=localhost:2082 \
  LOCKPROXY_DEBUGLISTENADDR=localhost:2083 \
  ./main ./dummycmd -addr localhost:2080

LOCKPROXY_UPSTREAMADDR=localhost:3080 \
  LOCKPROXY_PROXYLISTENADDR=localhost:3081 \
  LOCKPROXY_HEALTHLISTENADDR=localhost:3082 \
  LOCKPROXY_DEBUGLISTENADDR=localhost:3083 \
  ./main ./dummycmd -addr localhost:3080

grpc-health-probe -addr 127.0.0.1:1081
grpc-health-probe -addr 127.0.0.1:2081
grpc-health-probe -addr 127.0.0.1:3081

grpc-health-probe -addr 127.0.0.1:1082
grpc-health-probe -addr 127.0.0.1:2082
grpc-health-probe -addr 127.0.0.1:3082

grpc-health-probe -addr 127.0.0.1:1082 -service lockproxyleader
grpc-health-probe -addr 127.0.0.1:2082 -service lockproxyleader
grpc-health-probe -addr 127.0.0.1:3082 -service lockproxyleader

./dummyclient -addr 127.0.0.1:1081 lockproxy test
./dummyclient -addr 127.0.0.1:2081 lockproxy test
./dummyclient -addr 127.0.0.1:3081 lockproxy test
```

## Testing

```sh
etcd

docker run --rm -p 6381:6379 redis

export ETCD_ENDPOINT="localhost:2379"
export REDIS_ADDR="localhost:6381"

go test -p=1 ./...
```

### Coverage

```sh
go test -p=1 \
  -coverprofile lockproxy.coverprofile \
  -coverpkg ./pkg/lockproxy,./pkg/lockproxy/etcdadapter,./pkg/lockproxy/redisadapter \
  ./... \
  && go tool cover -html=lockproxy.coverprofile -o lockproxy.coverprofile.html
```

## Debug

Get a list of goroutines:

```sh
curl localhost:4083/debug/pprof/goroutine?debug=2
```
