package httpserver_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
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

	addr := freeAddr(t)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := httpserver.New(httpserver.Options{
		Addr:    addr,
		Handler: mux,
		Logger:  log,
	})
	readiness := httpserver.NewReadiness()
	readiness.Set(true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- httpserver.Serve(ctx, srv, log, readiness)
	}()

	waitUntilListening(t, addr)

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

	// Readiness must clear before drain completes so probes fail under load.
	waitUntil(t, time.Second, func() bool { return !readiness.Ready() })

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
	err := httpserver.Serve(context.Background(), nil, slog.Default(), nil)
	if err == nil {
		t.Fatal("expected error for nil server")
	}
}

func TestLiveHandler_AlwaysOK(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	httpserver.LiveHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}

func TestReadyHandler_ProcessGateAndChecks(t *testing.T) {
	t.Parallel()

	gate := httpserver.NewReadiness()
	var checkCalls atomic.Int32
	checkOK := func(context.Context) error {
		checkCalls.Add(1)
		return nil
	}
	checkFail := func(context.Context) error {
		checkCalls.Add(1)
		return errors.New("apiserver down")
	}

	t.Run("not ready before Set", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpserver.ReadyHandler(gate, checkOK).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
		if checkCalls.Load() != 0 {
			t.Fatal("dependency check must not run when process gate is closed")
		}
	})

	gate.Set(true)

	t.Run("ready when gate open and checks pass", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpserver.ReadyHandler(gate, checkOK).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("not ready when dependency fails", func(t *testing.T) {
		rec := httptest.NewRecorder()
		httpserver.ReadyHandler(gate, checkFail).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
	})

	t.Run("not ready after gate closed", func(t *testing.T) {
		gate.Set(false)
		rec := httptest.NewRecorder()
		httpserver.ReadyHandler(gate, checkOK).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", rec.Code)
		}
	})
}

func TestRegisterProbes_Routes(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	gate := httpserver.NewReadiness()
	httpserver.RegisterProbes(mux, gate, func(context.Context) error { return nil })

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	resp, err := http.Get(srv.URL + "/healthz") //nolint:noctx // test helper
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/healthz status = %d, want 200", resp.StatusCode)
	}

	resp, err = http.Get(srv.URL + "/readyz") //nolint:noctx // test helper
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("/readyz before Set status = %d, want 503", resp.StatusCode)
	}

	gate.Set(true)
	resp, err = http.Get(srv.URL + "/readyz") //nolint:noctx // test helper
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/readyz after Set status = %d, want 200", resp.StatusCode)
	}
}

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}

func waitUntilListening(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		conn, dialErr := net.DialTimeout("tcp", addr, 50*time.Millisecond)
		if dialErr == nil {
			_ = conn.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("server never became reachable on %s: %v", addr, dialErr)
		}
		time.Sleep(10 * time.Millisecond)
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
