package main

import (
	"context"
	"fmt"
	"net/http"
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
