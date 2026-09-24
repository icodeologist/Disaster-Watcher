package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
)

// drainStage describes one pipeline boundary. Closing input tells that stage
// there will be no more work. Waiting before closing the next stage guarantees
// that every output produced by this stage can still be delivered downstream.
type drainStage struct {
	name    string
	input   chanCloser
	workers *sync.WaitGroup
}

type chanCloser func()

type shutdownPipeline struct {
	recoveryWorkers *sync.WaitGroup
	stages          []drainStage
	closeAfterDrain chanCloser
}

func watchForForcedShutdown(signalChannel <-chan os.Signal, shutdownCtx context.Context, shutdownComplete <-chan struct{}, cancelWorkers context.CancelFunc, cancelShutdown context.CancelFunc) {
	select {
	case secondSignal, ok := <-signalChannel:
		if !ok {
			return
		}
		slog.Warn("Received second shutdown signal; stopping immediately", "signal", secondSignal)
		cancelWorkers()
		cancelShutdown()
	case <-shutdownCtx.Done():
		cancelWorkers()
	case <-shutdownComplete:
	}
}

// shutdownGracefully first stops HTTP intake and waits for active handlers.
// It then drains the worker pipeline from upstream to downstream.
func shutdownGracefully(ctx context.Context, server *http.Server, pipeline shutdownPipeline) error {
	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("stop HTTP server: %w", err)
	}

	if pipeline.recoveryWorkers != nil {
		if err := waitForWorkers(ctx, pipeline.recoveryWorkers); err != nil {
			return fmt.Errorf("wait for startup recovery: %w", err)
		}
	}

	for _, stage := range pipeline.stages {
		stage.input()
		if err := waitForWorkers(ctx, stage.workers); err != nil {
			return fmt.Errorf("drain %s work: %w", stage.name, err)
		}
	}

	if pipeline.closeAfterDrain != nil {
		pipeline.closeAfterDrain()
	}
	return nil
}
