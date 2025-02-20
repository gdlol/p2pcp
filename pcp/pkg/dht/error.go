package dht

import (
	"context"
	"fmt"
	"log/slog"
)

type ErrConnThresholdNotReached struct {
	BootstrapErrs []error
}

func (e ErrConnThresholdNotReached) Error() string {
	return "could not establish enough connections to bootstrap peers"
}

func (e ErrConnThresholdNotReached) Log() {
	// If only one error is context.Canceled the user stopped the
	// program and we don't want to print errors.
	for _, err := range e.BootstrapErrs {
		if err == context.Canceled {
			return
		}
	}

	slog.Warn(e.Error())
	for _, err := range e.BootstrapErrs {
		slog.Warn(fmt.Sprintf("\t%s\n", err))
	}

	slog.Warn("this means you will only be able to transfer files in your local network")
}
