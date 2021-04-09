package context

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/assetnote/kiterunner/pkg/log"
)

var (
	ctx            context.Context
	cancel         context.CancelFunc
	ctxInitialized sync.Once
)

// AddInterruptCancellation will add an interrupt handler that will catch the first SIGTERM and cancel the context
// upon second SIGTERM, the program will exit immediately
// This wrapping allows for graceful shutdown of the application
func AddInterruptCancellation(ctx context.Context, cancel context.CancelFunc) {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		interrupts := 0
		for {
			select {
			case <-c:
				interrupts++
				if interrupts > 1 {
					log.Info().Msg("Received multiple interrupt signals. Exiting")
					os.Exit(1)
				}
				log.Info().Msg("Received interrupt signal")
				cancel()
			case <-ctx.Done():
			}
		}
	}()
}

// InitContext will initialize the global context used to catch interrupts. This is automatically called
// by Context and Cancel
func InitContext() {
	ctxInitialized.Do(func() {
		ctx, cancel = context.WithCancel(context.Background())
		AddInterruptCancellation(ctx, cancel)
	})
}

// Context will initialize the global context and attach the interrupt handler that will cancel the context
// upon SIGTERM. This is safe to call from multiple goroutines and will always return the same context
func Context() context.Context {
	InitContext()
	return ctx
}

// Cancel will cancel the global context. Calling this multiple times is the equivalent of cancelling
// the same context multiple times
func Cancel() {
	InitContext()
	cancel()
}

