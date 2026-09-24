package cache

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) count(substr string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.Count(b.buf.String(), substr)
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestRedisStore_MonitorLogsOutageAndRecovery(t *testing.T) {
	mr := miniredis.RunT(t)
	store, err := NewRedisStore(&RedisStoreConfig{URL: "redis://" + mr.Addr(), OpTimeout: 20 * time.Millisecond})
	if err != nil {
		t.Fatalf("NewRedisStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	logs := &syncBuffer{}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- store.Monitor(ctx, &MonitorConfig{
			Logger:      slog.New(slog.NewTextHandler(logs, nil)),
			Interval:    10 * time.Millisecond,
			RepeatEvery: 60 * time.Millisecond,
		})
	}()

	time.Sleep(50 * time.Millisecond)
	if logs.count("Redis") != 0 {
		t.Fatal("a healthy Redis should log nothing")
	}

	mr.Close()
	waitFor(t, "the outage warning", func() bool { return logs.count("Redis unreachable") == 1 })
	waitFor(t, "a repeated warning", func() bool { return logs.count("Redis still unreachable") >= 1 })

	if err := mr.Restart(); err != nil {
		t.Fatalf("restart miniredis: %v", err)
	}
	waitFor(t, "the recovery log", func() bool { return logs.count("Redis reachable again") == 1 })

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Monitor: %v", err)
	}
	if logs.count("Redis unreachable") != 1 {
		t.Fatal("the outage should be announced once, then repeated as 'still unreachable'")
	}
}

func TestRedisStore_MonitorRequiresLogger(t *testing.T) {
	mr := miniredis.RunT(t)
	store, _ := NewRedisStore(&RedisStoreConfig{URL: "redis://" + mr.Addr()})
	t.Cleanup(func() { _ = store.Close() })
	if err := store.Monitor(context.Background(), nil); err == nil {
		t.Fatal("a monitor with nowhere to log should be rejected")
	}
}
