package lockproxy

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// Commander starts the command.
// It handles graceful shutdown with a timeout after which the command
// is forcefully killed.
type Commander struct {
	cmd                []string
	cmdShutdownTimeout time.Duration
	logger             *logrus.Entry
}

func NewCommander(
	cmd []string,
	cmdShutdownTimeout time.Duration,
	logger *logrus.Entry,
) *Commander {
	logger = logger.WithFields(logrus.Fields{
		"cmd": strings.Join(cmd, " "),
	})

	return &Commander{
		cmd:                cmd,
		cmdShutdownTimeout: cmdShutdownTimeout,
		logger:             logger,
	}
}

func (c *Commander) Start(ctx context.Context) error {
	cmdCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cmdExited := make(chan struct{}, 1)

	c.logger.Info("Commander starting command")

	cmd := exec.CommandContext(cmdCtx, c.cmd[0], c.cmd[1:]...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	go func() {
		select {
		case <-ctx.Done():
			c.logger.WithFields(logrus.Fields{
				"cmdShutdownTimeout": c.cmdShutdownTimeout,
			}).Info("Commander gracefully shutting down")

			_ = cmd.Process.Signal(syscall.SIGINT)

			timer := time.NewTimer(c.cmdShutdownTimeout)

			select {
			case <-timer.C:
				c.logger.Warn("Commander forcefully shutting down")
				cancel()
			case <-cmdExited:
				timer.Stop()
			}
		case <-cmdExited:
		}
	}()

	err := cmd.Run()

	cmdExited <- struct{}{}

	if err == nil {
		c.logger.Info("Commander command successfully exited")
	} else {
		c.logger.WithError(err).Info("Commander command exited with error")
	}

	return err
}
