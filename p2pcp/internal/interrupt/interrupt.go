package interrupt

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
)

var lock sync.Once

func RegisterInterruptHandler(ctx context.Context, handler func()) {
	lock.Do(func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)
		go func() {
			count := 0
			for {
				select {
				case <-ctx.Done():
					return
				case <-sigChan:
					count++
					if count == 1 {
						fmt.Println("Canceling...")
						go handler()
					} else {
						os.Exit(1)
					}
				}
			}
		}()
	})
}
