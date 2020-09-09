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

go build ./pkg/lockproxy/dummycmd
go build main.go

export LOCKPROXY_LOGLEVEL="debug"

# etcd
export LOCKPROXY_ADAPTER="etcd"
export LOCKPROXY_ETCDLOCKKEY="/applock"
export LOCKPROXY_ETCDADDRKEY="/leaderaddr"
export LOCKPROXY_LOGLEVEL="debug"

# redis
export LOCKPROXY_ADAPTER="redis"
export LOCKPROXY_REDISURL="redis://localhost:6381"
export LOCKPROXY_REDISLOCKKEY="applock"
export LOCKPROXY_REDISADDRKEY="leaderaddr"

UPSTREAMCMD_PORT=1080 \
  LOCKPROXY_CMD="./dummycmd -addr localhost:1080" \
  LOCKPROXY_UPSTREAMADDR=localhost:1080 \
  LOCKPROXY_PROXYLISTENADDR=localhost:1081 \
  LOCKPROXY_HEALTHLISTENADDR=localhost:1082 \
  LOCKPROXY_DEBUGLISTENADDR=localhost:1083 \
  ./main

UPSTREAMCMD_PORT=2080 \
  LOCKPROXY_CMD="./dummycmd -addr localhost:2080" \
  LOCKPROXY_UPSTREAMADDR=localhost:2080 \
  LOCKPROXY_PROXYLISTENADDR=localhost:2081 \
  LOCKPROXY_HEALTHLISTENADDR=localhost:2082 \
  LOCKPROXY_DEBUGLISTENADDR=localhost:2083 \
  ./main

UPSTREAMCMD_PORT=3080 \
  LOCKPROXY_CMD="./dummycmd -addr localhost:3080" \
  LOCKPROXY_UPSTREAMADDR=localhost:3080 \
  LOCKPROXY_PROXYLISTENADDR=localhost:3081 \
  LOCKPROXY_HEALTHLISTENADDR=localhost:3082 \
  LOCKPROXY_DEBUGLISTENADDR=localhost:3083 \
  ./main

grpc-health-probe -addr 127.0.0.1:1081
grpc-health-probe -addr 127.0.0.1:2081
grpc-health-probe -addr 127.0.0.1:3081
```

## Testing

```sh
etcd

docker run --rm -p 6381:6379 redis

export ETCD_ENDPOINT="localhost:2379"
export REDIS_ADDR="localhost:6381"

go test ./...
```

### Coverage

```sh
go test --coverprofile lockproxy.coverprofile ./pkg/lockproxy && go tool cover -html=lockproxy.coverprofile -o lockproxy.coverprofile.html
```

## Debug

Get a list of goroutines:

```sh
curl localhost:4083/debug/pprof/goroutine?debug=2
```
