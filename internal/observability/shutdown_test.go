package observability

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

func testLogger() *slog.Logger {
	logger, _ := NewJSONLogger("error", io.Discard)
	return logger
}

func TestShutdownHTTPServerWaitsForInFlightRequest(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var canceled atomic.Bool
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, _ = io.WriteString(w, "completed")
	})}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()

	responseDone := make(chan error, 1)
	go func() {
		response, err := http.Get("http://" + listener.Addr().String())
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
		responseDone <- err
	}()
	<-entered

	shutdownStarted := make(chan struct{})
	server.RegisterOnShutdown(func() { close(shutdownStarted) })
	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- ShutdownHTTPServer(server, time.Second, func() { canceled.Store(true) }, testLogger())
	}()
	<-shutdownStarted
	close(release)
	if err := <-shutdownDone; err != nil {
		t.Fatalf("graceful shutdown failed: %v", err)
	}
	if err := <-responseDone; err != nil {
		t.Fatalf("in-flight request failed: %v", err)
	}
	if !canceled.Load() {
		t.Fatal("application context was not canceled after draining")
	}
	if err := <-serveDone; !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("unexpected Serve result: %v", err)
	}
}

func TestShutdownHTTPServerForcesCloseAfterDeadline(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release
		_, _ = io.WriteString(w, "eventually completed")
	})}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	clientDone := make(chan struct{})
	go func() {
		defer close(clientDone)
		response, err := http.Get("http://" + listener.Addr().String())
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
	}()
	<-entered

	var canceled atomic.Bool
	shutdownStarted := make(chan struct{})
	server.RegisterOnShutdown(func() { close(shutdownStarted) })
	shutdownDone := make(chan error, 1)
	start := time.Now()
	go func() {
		shutdownDone <- ShutdownHTTPServer(server, 20*time.Millisecond, func() { canceled.Store(true) }, testLogger())
	}()
	<-shutdownStarted
	err = <-shutdownDone
	if err == nil {
		t.Fatal("expected graceful shutdown deadline error")
	}
	if time.Since(start) > time.Second {
		t.Fatalf("shutdown exceeded bounded deadline: %s", time.Since(start))
	}
	if !canceled.Load() {
		t.Fatal("application context was not canceled after deadline")
	}
	close(release)
	<-clientDone
	if serveErr := <-serveDone; !errors.Is(serveErr, http.ErrServerClosed) {
		t.Fatalf("unexpected Serve result: %v", serveErr)
	}
}

func TestShutdownHTTPServerUsesFallbackForNonPositiveTimeout(t *testing.T) {
	server := &http.Server{Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	if err := ShutdownHTTPServer(server, 0, func() {}, testLogger()); err != nil {
		t.Fatal(err)
	}
	if err := <-serveDone; !errors.Is(err, http.ErrServerClosed) {
		t.Fatalf("unexpected Serve result: %v", err)
	}
}
