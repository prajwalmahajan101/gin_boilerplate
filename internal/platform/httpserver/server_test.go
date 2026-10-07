package httpserver

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prajwalmahajan101/gin_boilerplate/internal/config"
)

func TestServer_GracefulShutdownOnContextCancel(t *testing.T) {
	r := gin.New()
	r.GET("/healthz", Liveness())

	// Port 0 lets the OS pick a free port, but http.Server needs a known addr
	// to probe; bind explicitly to an ephemeral loopback port.
	cfg := &config.Config{Port: "0", ShutdownTimeoutS: 5}
	srv := New(cfg, r)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.RunWithGracefulShutdown(ctx) }()

	// Give the listener a moment to come up, then trigger shutdown.
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunWithGracefulShutdown returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down within 2s")
	}
}

func TestServer_PortBindingFailureReturnsError(t *testing.T) {
	// An out-of-range port surfaces the ListenAndServe error promptly.
	bad := New(&config.Config{Port: "999999", ShutdownTimeoutS: 1}, gin.New())
	if err := bad.RunWithGracefulShutdown(context.Background()); err == nil {
		t.Fatal("expected error for invalid port, got nil")
	}
}
