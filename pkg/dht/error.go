package dht

import (
	"context"

	"p2pcp/internal/log"
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

	log.Warningln(e)
	for _, err := range e.BootstrapErrs {
		log.Warningf("\t%s\n", err)
	}

	log.Warningln("this means you will only be able to transfer files in your local network")
}
