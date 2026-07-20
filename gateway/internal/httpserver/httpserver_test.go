package httpserver_test

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eu-sovereign-cloud/ecp/gateway/internal/httpserver"
)

func TestServe_GracefulShutdownDrainsInFlight(t *testing.T) {
	t.Parallel()

	var entered atomic.Bool
	release := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		entered.Store(true)
		select {
		case <-release:
		case <-r.Context().Done():
			http.Error(w, "cancelled", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httpserver.New(httpserver.Options{
		Addr:    addr,
		Handler: mux,
		Logger:  log,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- httpserver.Serve(ctx, srv, log)
	}()

	// Wait until the server accepts connections.
	deadline := time.Now().Add(2 * time.Second)
	for {
		conn, dialErr := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("server never became reachable: %v", dialErr)
		}
		time.Sleep(10 * time.Millisecond)
	}

	clientDone := make(chan struct{})
	var clientStatus int
	var clientBody string
	go func() {
		defer close(clientDone)
		resp, getErr := http.Get("http://" + addr + "/slow") //nolint:noctx // test helper
		if getErr != nil {
			t.Errorf("client get: %v", getErr)
			return
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		clientStatus = resp.StatusCode
		clientBody = string(b)
	}()

	// Wait until the handler is in-flight, then cancel serve context.
	waitUntil(t, 2*time.Second, entered.Load)
	cancel()

	// Give Shutdown a moment to start draining, then release the handler.
	time.Sleep(50 * time.Millisecond)
	close(release)

	select {
	case <-clientDone:
	case <-time.After(httpserver.DefaultShutdownTimeout + time.Second):
		t.Fatal("client request did not complete during graceful shutdown")
	}

	if clientStatus != http.StatusOK || clientBody != "ok" {
		t.Fatalf("client got status=%d body=%q, want 200/ok", clientStatus, clientBody)
	}

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("Serve returned error: %v", err)
		}
	case <-time.After(httpserver.DefaultShutdownTimeout + time.Second):
		t.Fatal("Serve did not return after shutdown")
	}
}

func TestServe_NilServer(t *testing.T) {
	t.Parallel()
	err := httpserver.Serve(context.Background(), nil, slog.Default())
	if err == nil {
		t.Fatal("expected error for nil server")
	}
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met before timeout")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
