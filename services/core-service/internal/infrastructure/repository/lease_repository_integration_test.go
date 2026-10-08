//go:build integration

package repository

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
)

// leaseTestDB connects with clientFoundRows, which makes an upsert that changes nothing still report
// its matched row. Production databases report rows affected that way, so Acquire must not rely on it.
func leaseTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("SQL_PREPARE_TEST_DSN")
	if dsn == "" {
		dsn = "root:Testing123!@tcp(localhost:3306)/openmrp?parseTime=true"
	}
	pool, err := sql.Open("mysql", dsn+"&clientFoundRows=true")
	if err != nil {
		t.Fatalf("open mysql: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := pool.PingContext(ctx); err != nil {
		t.Fatalf("ping mysql: %v", err)
	}
	t.Cleanup(func() { _ = pool.Close() })
	return pool
}

func TestLeaseAcquire_HeldLeaseExcludesOtherHolders(t *testing.T) {
	ctx := context.Background()
	pool := leaseTestDB(t)
	repo := NewLeaseRepo(sqlc.New(pool))

	name := "lease_test_" + time.Now().Format("150405.000000000")
	t.Cleanup(func() { _, _ = pool.Exec("DELETE FROM task_leases WHERE name = ?", name) })

	acquire := func(holder string, ttl time.Duration) bool {
		t.Helper()
		ok, err := repo.Acquire(ctx, name, holder, ttl)
		if err != nil {
			t.Fatalf("acquire as %s: %v", holder, err)
		}
		return ok
	}

	if !acquire("pod-a", time.Minute) {
		t.Fatal("pod-a should acquire a free lease")
	}
	if acquire("pod-b", time.Minute) {
		t.Fatal("pod-b acquired a lease pod-a holds")
	}
	if !acquire("pod-a", time.Minute) {
		t.Fatal("pod-a should re-acquire its own lease")
	}

	if _, err := pool.Exec("UPDATE task_leases SET expires_at = NOW(6) - INTERVAL 1 SECOND WHERE name = ?", name); err != nil {
		t.Fatalf("expire lease: %v", err)
	}
	if !acquire("pod-b", time.Minute) {
		t.Fatal("pod-b should take an expired lease")
	}
	if acquire("pod-a", time.Minute) {
		t.Fatal("pod-a acquired a lease pod-b now holds")
	}
}
