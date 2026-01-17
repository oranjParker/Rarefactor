package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping main integration test in short mode")
	}

	// Set test environment variables to point to real services if available,
	// or use ports that are unlikely to be used.
	t.Setenv("GRPC_PORT", "50052")
	t.Setenv("HTTP_PORT", "8001")

	// Ensure we have some basic environment variables set if not already
	if os.Getenv("DATABASE_URL") == "" {
		t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/rarefactor?sslmode=disable")
	}
	if os.Getenv("REDIS_URL") == "" {
		t.Setenv("REDIS_URL", "redis://localhost:6379")
	}
	if os.Getenv("QDRANT_URL") == "" {
		t.Setenv("QDRANT_URL", "localhost:6334")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := run(ctx)
	if err != nil && !strings.Contains(err.Error(), "connection refused") && !strings.Contains(err.Error(), "context canceled") && !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("run() returned unexpected error: %v", err)
	}
}

func TestRunWithDeps(t *testing.T) {
	t.Setenv("GRPC_PORT", "50053")
	t.Setenv("HTTP_PORT", "8002")

	deps := &AppDependencies{
		Pool:     nil, // These can be nil as per our NewSearchServer/NewCrawlerServer usage in runWithDeps
		Redis:    nil,
		Qdrant:   nil,
		Embedder: nil,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := runWithDeps(ctx, deps)
	if err != nil && !strings.Contains(err.Error(), "context canceled") && !strings.Contains(err.Error(), "deadline exceeded") {
		t.Fatalf("runWithDeps() returned unexpected error: %v", err)
	}
}

func TestRun_Error(t *testing.T) {
	t.Setenv("DATABASE_URL", "invalid_url")
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := run(ctx)
	if err == nil {
		t.Error("expected error due to invalid DATABASE_URL, got nil")
	}
}

func TestRunWithDeps_ListenError(t *testing.T) {
	t.Setenv("GRPC_PORT", "-1") // Invalid port
	deps := &AppDependencies{}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := runWithDeps(ctx, deps)
	if err == nil {
		t.Error("expected error due to invalid GRPC_PORT, got nil")
	}
}
