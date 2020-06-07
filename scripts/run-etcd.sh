#!/bin/sh

export PATH="$PATH:/tmp/etcd"

if ! [ -x "$(command -v etcd)" ]; then
  echo 'Error: etcd is not installed.' >&2
  exit 1
fi

if ! [ -x "$(command -v etcdctl)" ]; then
  echo 'Error: etcdctl is not installed.' >&2
  exit 1
fi

etcd &

while true; do
  if etcdctl endpoint status; then
    break
  fi
  echo 'Waiting for etcd to start'
  sleep 1
done
