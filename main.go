package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"

	"github.com/bancek/lockproxy/pkg/lockproxy"
	"github.com/bancek/lockproxy/pkg/lockproxy/config"
	"github.com/bancek/lockproxy/pkg/lockproxy/etcdadapter"
	"github.com/bancek/lockproxy/pkg/lockproxy/redisadapter"
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

		checkEnvConfig(cfg, etcdCfg, logger)

		adapter = etcdadapter.NewEtcdAdapter(etcdCfg, logger)
	} else if cfg.Adapter == "redis" {
		redisCfg := &redisadapter.RedisConfig{}

		err := envconfig.Process("lockproxy", redisCfg)
		if err != nil {
			logger.WithError(err).Fatal("Failed to load redis config from env")
		}

		checkEnvConfig(cfg, redisCfg, logger)

		adapter = redisadapter.NewRedisAdapter(redisCfg, logger)
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

func checkEnvConfig(cfg *config.Config, adapterCfg interface{}, logger *logrus.Entry) {
	adapterName := strings.ToUpper(cfg.Adapter)

	prefix := "LOCKPROXY_"
	adapterPrefix := prefix + adapterName

	checkPrefix := strings.ToUpper(uuid.New().String())
	adapterCheckPrefix := strings.ToUpper(uuid.New().String())

	setKeys := []string{}

	for _, env := range os.Environ() {
		if !strings.HasPrefix(env, prefix) {
			continue
		}
		keyVal := strings.SplitN(env, "=", 2)
		key := keyVal[0]
		val := ""
		if len(keyVal) > 1 {
			val = keyVal[1]
		}
		checkKey := checkPrefix + "_" + key
		if strings.HasPrefix(key, adapterPrefix) {
			checkKey = adapterCheckPrefix + "_" + key
		}
		_ = os.Setenv(checkKey, val)
		setKeys = append(setKeys, checkKey)
	}

	defer func() {
		for _, key := range setKeys {
			_ = os.Unsetenv(key)
		}
	}()

	if checkErr := envconfig.CheckDisallowed(checkPrefix+"_LOCKPROXY", cfg); checkErr != nil {
		logger.WithError(errors.New(strings.ReplaceAll(checkErr.Error(), checkPrefix+"_", ""))).Warn("Config check error")
	}

	if checkErr := envconfig.CheckDisallowed(adapterCheckPrefix+"_LOCKPROXY", adapterCfg); checkErr != nil {
		logger.WithError(errors.New(strings.ReplaceAll(checkErr.Error(), adapterCheckPrefix+"_", ""))).Warn("Config check error")
	}
}
