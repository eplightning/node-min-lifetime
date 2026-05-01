package supervisor

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func SignalContext() context.Context {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-sigCh
		cancel()
	}()

	return ctx
}
