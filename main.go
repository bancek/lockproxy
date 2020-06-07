package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/bancek/lockproxy/lockproxy"
	"github.com/kelseyhightower/envconfig"
	"github.com/sirupsen/logrus"
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

	config := lockproxy.Config{}

	err := envconfig.Process("lockproxy", &config)
	if err != nil {
		logger.Fatal(err)
	}

	logLevel, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logger.Fatal(err)
	}
	baseLogger.SetLevel(logLevel)

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

	proxy := lockproxy.NewLockProxy(&config, logger)

	err = proxy.Init(ctx)
	if err != nil {
		os.Exit(1)
	}
	defer proxy.Close()

	err = proxy.Start()
	if err != nil {
		os.Exit(2)
	}
}
