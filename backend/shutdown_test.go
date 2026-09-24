package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type shutdownReport struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

type shutdownState struct {
	mu     sync.Mutex
	report shutdownReport
	status string
	events []string
}

func (s *shutdownState) event(name string) {
	s.mu.Lock()
	s.events = append(s.events, name)
	s.mu.Unlock()
}

type simplePipeline struct {
	shutdownPipeline
	verification chan int64
	reports      chan int64
	delivery     chan int64
	retry        chan int64
	deadLetters  chan int64
}

func newSimplePipeline(state *shutdownState, workerStarted, releaseWorker chan struct{}) simplePipeline {
	p := simplePipeline{
		verification: make(chan int64, 1),
		reports:      make(chan int64, 1),
		delivery:     make(chan int64, 1),
		retry:        make(chan int64, 1),
		deadLetters:  make(chan int64, 1),
	}

	var verificationWG, extractionWG, notificationWG, retryWG sync.WaitGroup
	verificationWG.Add(1)
	go func() {
		defer verificationWG.Done()
		defer state.event("verification workers done")
		for jobID := range p.verification {
			close(workerStarted)
			<-releaseWorker
			p.reports <- jobID
		}
	}()

	startStage := func(name string, input <-chan int64, output chan<- int64, wg *sync.WaitGroup) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer state.event(name + " workers done")
			for jobID := range input {
				if output != nil {
					output <- jobID
				}
			}
		}()
	}
	startStage("extraction", p.reports, p.delivery, &extractionWG)
	startStage("notification", p.delivery, p.retry, &notificationWG)

	retryWG.Add(1)
	go func() {
		defer retryWG.Done()
		defer state.event("retry workers done")
		for range p.retry {
			state.mu.Lock()
			state.status = "done"
			state.mu.Unlock()
		}
	}()

	closeStage := func(name string, channel chan int64) chanCloser {
		return func() {
			state.event("close " + name)
			close(channel)
		}
	}
	p.shutdownPipeline = shutdownPipeline{
		stages: []drainStage{
			{name: "verification", input: closeStage("verification", p.verification), workers: &verificationWG},
			{name: "reports", input: closeStage("reports", p.reports), workers: &extractionWG},
			{name: "delivery", input: closeStage("delivery", p.delivery), workers: &notificationWG},
			{name: "retry", input: closeStage("retry", p.retry), workers: &retryWG},
		},
		closeAfterDrain: closeStage("dead-letter", p.deadLetters),
	}
	return p
}

func reportServer(t *testing.T, state *shutdownState, verification chan<- int64, handlerOverride http.Handler) (*http.Server, string) {
	t.Helper()
	handler := http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		var report shutdownReport
		if err := json.NewDecoder(request.Body).Decode(&report); err != nil {
			http.Error(response, "invalid report", http.StatusBadRequest)
			return
		}
		report.ID = 1
		state.mu.Lock()
		state.report = report
		state.status = "processing"
		state.mu.Unlock()
		verification <- report.ID
		response.WriteHeader(http.StatusAccepted)
	})
	if handlerOverride != nil {
		handler = handlerOverride.ServeHTTP
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	server := &http.Server{Handler: handler}
	serveDone := make(chan struct{})
	go func() {
		defer close(serveDone)
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = server.Close()
		<-serveDone
	})
	return server, "http://" + listener.Addr().String()
}

func waitShutdown(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("shutdown failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("shutdown did not finish")
	}
}

func assertClosed(t *testing.T, name string, channel <-chan int64) {
	t.Helper()
	if jobID, open := <-channel; open {
		t.Fatalf("%s still contained job %d after shutdown", name, jobID)
	}
}

func TestShutdownDrainsAcceptedReport(t *testing.T) {
	state := &shutdownState{}
	workerStarted := make(chan struct{})
	releaseWorker := make(chan struct{})
	pipeline := newSimplePipeline(state, workerStarted, releaseWorker)
	server, baseURL := reportServer(t, state, pipeline.verification, nil)

	response, err := http.Post(baseURL, "application/json", strings.NewReader(`{"title":"Flood warning"}`))
	if err != nil {
		t.Fatalf("post report: %v", err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusAccepted)
	}
	<-workerStarted

	done := make(chan error, 1)
	go func() { done <- shutdownGracefully(context.Background(), server, pipeline.shutdownPipeline) }()
	select {
	case err := <-done:
		t.Fatalf("shutdown finished while worker was busy: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	close(releaseWorker)
	waitShutdown(t, done)

	state.mu.Lock()
	report, status, events := state.report, state.status, append([]string(nil), state.events...)
	state.mu.Unlock()
	if report.Title != "Flood warning" || report.ID != 1 {
		t.Fatalf("stored report = %+v, want accepted report", report)
	}
	if status != "done" {
		t.Fatalf("status = %q, want done", status)
	}
	assertClosed(t, "verification", pipeline.verification)
	assertClosed(t, "reports", pipeline.reports)
	assertClosed(t, "delivery", pipeline.delivery)
	assertClosed(t, "retry", pipeline.retry)
	assertClosed(t, "dead-letter", pipeline.deadLetters)

	wantOrder := []string{
		"close verification", "verification workers done",
		"close reports", "extraction workers done",
		"close delivery", "notification workers done",
		"close retry", "retry workers done",
		"close dead-letter",
	}
	for i, want := range wantOrder {
		if i >= len(events) || events[i] != want {
			t.Fatalf("events = %v, want prefix %v", events, wantOrder)
		}
	}
}

func TestShutdownWaitsForActiveReportHandler(t *testing.T) {
	state := &shutdownState{}
	workerStarted := make(chan struct{})
	releaseWorker := make(chan struct{})
	pipeline := newSimplePipeline(state, workerStarted, releaseWorker)
	handlerStarted := make(chan struct{})
	releaseHandler := make(chan struct{})
	handler := http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		close(handlerStarted)
		<-releaseHandler
		state.mu.Lock()
		state.report = shutdownReport{ID: 1, Title: "Flood warning"}
		state.status = "processing"
		state.mu.Unlock()
		pipeline.verification <- 1
		response.WriteHeader(http.StatusAccepted)
	})
	server, baseURL := reportServer(t, state, pipeline.verification, handler)

	requestDone := make(chan error, 1)
	go func() {
		response, err := http.Post(baseURL, "application/json", strings.NewReader(`{}`))
		if err == nil {
			response.Body.Close()
			if response.StatusCode != http.StatusAccepted {
				err = fmt.Errorf("status = %d, want %d", response.StatusCode, http.StatusAccepted)
			}
		}
		requestDone <- err
	}()
	<-handlerStarted

	done := make(chan error, 1)
	go func() { done <- shutdownGracefully(context.Background(), server, pipeline.shutdownPipeline) }()
	select {
	case err := <-done:
		t.Fatalf("shutdown finished before active handler completed: %v", err)
	case <-time.After(25 * time.Millisecond):
	}

	close(releaseHandler)
	if err := <-requestDone; err != nil {
		t.Fatalf("active request failed: %v", err)
	}
	<-workerStarted
	close(releaseWorker)
	waitShutdown(t, done)
}

func TestReportAfterShutdownIsRejected(t *testing.T) {
	state := &shutdownState{}
	workerStarted := make(chan struct{})
	releaseWorker := make(chan struct{})
	pipeline := newSimplePipeline(state, workerStarted, releaseWorker)
	server, baseURL := reportServer(t, state, pipeline.verification, nil)

	done := make(chan error, 1)
	go func() { done <- shutdownGracefully(context.Background(), server, pipeline.shutdownPipeline) }()
	waitShutdown(t, done)

	response, err := http.Post(baseURL, "application/json", strings.NewReader(`{"title":"Too late"}`))
	if err == nil {
		response.Body.Close()
		t.Fatalf("post after shutdown returned HTTP %d, want request failure", response.StatusCode)
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	if state.report.ID != 0 || state.status != "" {
		t.Fatalf("post after shutdown changed state: report=%+v status=%q", state.report, state.status)
	}
}

func TestShutdownWaitsForRecoveryBeforeClosingPipeline(t *testing.T) {
	recoveryStarted := make(chan struct{})
	releaseRecovery := make(chan struct{})
	verification := make(chan int64)
	deadLetters := make(chan int64)
	var recoveryWG sync.WaitGroup
	var verificationWG sync.WaitGroup
	recoveryWG.Add(1)
	go func() {
		defer recoveryWG.Done()
		close(recoveryStarted)
		<-releaseRecovery
	}()

	var eventsMu sync.Mutex
	var events []string
	recordEvent := func(event string) {
		eventsMu.Lock()
		events = append(events, event)
		eventsMu.Unlock()
	}
	pipeline := shutdownPipeline{
		recoveryWorkers: &recoveryWG,
		stages: []drainStage{
			{name: "verification", input: func() { recordEvent("close verification"); close(verification) }, workers: &verificationWG},
		},
		closeAfterDrain: func() { recordEvent("close dead-letter"); close(deadLetters) },
	}

	done := make(chan error, 1)
	go func() { done <- shutdownGracefully(context.Background(), &http.Server{}, pipeline) }()
	<-recoveryStarted

	select {
	case err := <-done:
		t.Fatalf("shutdown finished before recovery finished: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	eventsMu.Lock()
	if len(events) != 0 {
		t.Fatalf("shutdown closed channels before recovery finished: %v", events)
	}
	eventsMu.Unlock()

	close(releaseRecovery)
	waitShutdown(t, done)

	eventsMu.Lock()
	defer eventsMu.Unlock()
	if fmt.Sprint(events) != "[close verification close dead-letter]" {
		t.Fatalf("shutdown events = %v", events)
	}
}

func TestShutdownDrainsBufferedPipelineWork(t *testing.T) {
	const bufferedJobs = 4
	verification := make(chan int64, bufferedJobs)
	reports := make(chan int64, bufferedJobs)
	delivery := make(chan int64, bufferedJobs)
	retry := make(chan int64, bufferedJobs)
	deadLetters := make(chan int64, 1)
	processed := make(chan int64, bufferedJobs)

	startStage := func(input <-chan int64, output chan<- int64, wg *sync.WaitGroup) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for jobID := range input {
				if output == nil {
					processed <- jobID
					continue
				}
				output <- jobID
			}
		}()
	}

	var verificationWG, extractionWG, notificationWG, retryWG sync.WaitGroup
	startStage(verification, reports, &verificationWG)
	startStage(reports, delivery, &extractionWG)
	startStage(delivery, retry, &notificationWG)
	startStage(retry, nil, &retryWG)

	for jobID := int64(1); jobID <= bufferedJobs; jobID++ {
		verification <- jobID
	}

	pipeline := shutdownPipeline{
		stages: []drainStage{
			{name: "verification", input: func() { close(verification) }, workers: &verificationWG},
			{name: "reports", input: func() { close(reports) }, workers: &extractionWG},
			{name: "delivery", input: func() { close(delivery) }, workers: &notificationWG},
			{name: "retry", input: func() { close(retry) }, workers: &retryWG},
		},
		closeAfterDrain: func() { close(deadLetters) },
	}

	if err := shutdownGracefully(context.Background(), &http.Server{}, pipeline); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}

	seen := map[int64]bool{}
	for range bufferedJobs {
		select {
		case jobID := <-processed:
			seen[jobID] = true
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for buffered job")
		}
	}
	for jobID := int64(1); jobID <= bufferedJobs; jobID++ {
		if !seen[jobID] {
			t.Fatalf("buffered job %d was not processed; seen=%v", jobID, seen)
		}
	}
	assertClosed(t, "verification", verification)
	assertClosed(t, "reports", reports)
	assertClosed(t, "delivery", delivery)
	assertClosed(t, "retry", retry)
	assertClosed(t, "dead-letter", deadLetters)
}

func TestShutdownStopsWaitingWhenDeadlineExpires(t *testing.T) {
	verification := make(chan int64)
	releaseWorker := make(chan struct{})
	var verificationWG sync.WaitGroup
	verificationWG.Add(1)
	go func() {
		defer verificationWG.Done()
		<-releaseWorker
	}()

	pipeline := shutdownPipeline{
		stages: []drainStage{
			{name: "verification", input: func() { close(verification) }, workers: &verificationWG},
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	started := time.Now()
	err := shutdownGracefully(ctx, &http.Server{}, pipeline)
	if err == nil || !strings.Contains(err.Error(), "drain verification work") {
		t.Fatalf("shutdown error = %v, want verification drain timeout", err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("shutdown took %s after deadline, want bounded return", elapsed)
	}

	close(releaseWorker)
	verificationWG.Wait()
}
