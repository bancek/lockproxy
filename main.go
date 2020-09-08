package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	"github.com/bancek/lockproxy/pkg/lockproxy"
	"github.com/bancek/lockproxy/pkg/lockproxy/config"
	"github.com/bancek/lockproxy/pkg/lockproxy/etcdadapter"
)

func main() {
	baseLogger := logrus.StandardLogger()
	baseLogger.SetFormatter(&lockproxy.JSONFormatter{
		JSONFormatter: &logrus.JSONFormatter{
			TimestampFormat: lockproxy.RFC3339Milli,
		},
	})

	logger := baseLogger.WithFields(logrus.Fields{
		"service": "lockproxy",
	})

	cfg := &config.Config{}

	err := envconfig.Process("lockproxy", cfg)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load config from env")
	}

	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logger.WithError(err).Fatal("Failed to parse log level")
	}
	baseLogger.SetLevel(logLevel)

	if len(os.Args) > 1 {
		if len(cfg.Cmd) > 0 {
			logger.WithError(err).Fatal("Cannot specify both LOCKPROXY_CMD and process arguments")
		}

		cfg.Cmd = os.Args[1:]

		if cfg.Cmd[0] == "--" {
			cfg.Cmd = cfg.Cmd[1:]
		}
	}

	var adapter lockproxy.Adapter

	if cfg.Adapter == "etcd" {
		etcdCfg := &etcdadapter.EtcdConfig{}

		err := envconfig.Process("lockproxy", etcdCfg)
		if err != nil {
			logger.WithError(err).Fatal("Failed to load etcd config from env")
		}

		adapter = etcdadapter.NewEtcdAdapter(etcdCfg, logger)
	} else {
		logger.WithField("adapter", cfg.Adapter).Fatal("Invalid adapter")
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(interrupt)

	go func() {
		select {
		case <-interrupt:
			cancel()
		case <-ctx.Done():
		}
	}()

	proxy := lockproxy.NewLockProxy(cfg, adapter, logger)

	err = proxy.Init(ctx)
	if err != nil {
		logger.WithError(err).Fatal("Failed to init lock proxy")
	}
	defer proxy.Close()

	err = proxy.Start()
	if err != nil {
		logger.WithError(err).Warn("Lock proxy shutdown")
		os.Exit(2)
	}
}
