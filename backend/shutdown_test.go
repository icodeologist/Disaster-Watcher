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
