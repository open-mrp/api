//go:build integration

package repository

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/messaging"
)

// The inbox's two guarantees live in SQL, not in Go: Claim only takes a lease that is actually free,
// and Complete only marks a record that is still 'received'. Both are conditional UPDATEs whose
// behavior under concurrency is InnoDB's, so a mock cannot tell us whether they hold. These run
// against the real database.
//
//	go test -tags integration ./services/core-service/internal/infrastructure/repository/

func testDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("SQL_PREPARE_TEST_DSN")
	if dsn == "" {
		dsn = "root:Testing123!@tcp(localhost:3306)/openmrp?parseTime=true"
	}
	pool, err := sql.Open("mysql", dsn)
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

// insertRecord seeds one inbox row and removes it when the test ends.
func insertRecord(t *testing.T, pool *sql.DB, repo messaging.InboxRepo, handler string, ttlSeconds int) int64 {
	t.Helper()

	messageID := "msg_test_" + t.Name() + "_" + time.Now().Format("150405.000000000")
	id, err := repo.TryInsert(context.Background(), messaging.InboxRecordInput{
		MessageID:      messageID,
		ServiceName:    "core-service-test",
		Handler:        handler,
		MessageType:    "test.event",
		LockOwner:      "owner-a",
		LockTTLSeconds: ttlSeconds,
	})
	if err != nil {
		t.Fatalf("TryInsert: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec("DELETE FROM message_inbox WHERE id = ?", id)
	})
	return id
}

func TestInbox_TryInsertHoldsALease(t *testing.T) {
	pool := testDB(t)
	repo := NewInboxRepo(sqlc.New(pool))

	id := insertRecord(t, pool, repo, "test.lease_held", 300)

	var owner sql.NullString
	var expires sql.NullTime
	var status string
	row := pool.QueryRow("SELECT status, lock_owner, lock_expires_at FROM message_inbox WHERE id = ?", id)
	if err := row.Scan(&status, &owner, &expires); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if status != "received" {
		t.Errorf("status = %q, want received", status)
	}
	if owner.String != "owner-a" {
		t.Errorf("lock_owner = %q, want owner-a", owner.String)
	}
	if !expires.Valid || !expires.Time.After(time.Now()) {
		t.Errorf("lock_expires_at = %v, want a time in the future", expires.Time)
	}
}

// A live lease is what stops a redelivery running alongside the attempt that holds it.
func TestInbox_ClaimIsRefusedWhileTheLeaseIsLive(t *testing.T) {
	pool := testDB(t)
	repo := NewInboxRepo(sqlc.New(pool))

	id := insertRecord(t, pool, repo, "test.claim_refused", 300)

	claimed, err := repo.Claim(context.Background(), id, "owner-b", 300)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed {
		t.Error("Claim succeeded against a live lease; a redelivery would run alongside the live attempt")
	}
}

// Once the lease lapses the record is abandoned and the next attempt may take it.
func TestInbox_ClaimSucceedsOnceTheLeaseHasLapsed(t *testing.T) {
	pool := testDB(t)
	repo := NewInboxRepo(sqlc.New(pool))

	id := insertRecord(t, pool, repo, "test.claim_lapsed", 300)
	if _, err := pool.Exec(
		"UPDATE message_inbox SET lock_expires_at = DATE_SUB(NOW(3), INTERVAL 1 SECOND) WHERE id = ?", id,
	); err != nil {
		t.Fatalf("expiring the lease: %v", err)
	}

	claimed, err := repo.Claim(context.Background(), id, "owner-b", 300)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if !claimed {
		t.Fatal("Claim was refused on a lapsed lease; the message would never be retried")
	}

	var owner sql.NullString
	if err := pool.QueryRow("SELECT lock_owner FROM message_inbox WHERE id = ?", id).Scan(&owner); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if owner.String != "owner-b" {
		t.Errorf("lock_owner = %q, want the claiming consumer owner-b", owner.String)
	}
}

// A terminal record must never be claimable, however long ago its lease lapsed.
func TestInbox_ClaimIsRefusedOnTerminalRecords(t *testing.T) {
	pool := testDB(t)
	repo := NewInboxRepo(sqlc.New(pool))

	for _, status := range []string{"processed", "discarded"} {
		t.Run(status, func(t *testing.T) {
			id := insertRecord(t, pool, repo, "test.claim_terminal_"+status, 300)
			if _, err := pool.Exec(
				"UPDATE message_inbox SET status = ?, lock_expires_at = NULL, lock_owner = NULL WHERE id = ?",
				status, id,
			); err != nil {
				t.Fatalf("setting status: %v", err)
			}

			claimed, err := repo.Claim(context.Background(), id, "owner-b", 300)
			if err != nil {
				t.Fatalf("Claim: %v", err)
			}
			if claimed {
				t.Errorf("a %s record was claimable; its work would be applied again", status)
			}
		})
	}
}

func TestInbox_CompleteMarksAndReleases(t *testing.T) {
	pool := testDB(t)
	repo := NewInboxRepo(sqlc.New(pool))

	id := insertRecord(t, pool, repo, "test.complete", 300)

	completed, err := repo.Complete(context.Background(), id, "owner-a")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if !completed {
		t.Fatal("Complete matched no rows on a received record")
	}

	var status string
	var processedAt sql.NullTime
	var owner sql.NullString
	var expires sql.NullTime
	row := pool.QueryRow(
		"SELECT status, processed_at, lock_owner, lock_expires_at FROM message_inbox WHERE id = ?", id)
	if err := row.Scan(&status, &processedAt, &owner, &expires); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if status != "processed" {
		t.Errorf("status = %q, want processed", status)
	}
	if !processedAt.Valid {
		t.Error("processed_at was not stamped, so the retention purge will never collect the row")
	}
	if owner.Valid || expires.Valid {
		t.Error("the lease was not released on completion")
	}
}

// The second Complete is the losing side of a race. It must match nothing so the caller knows to roll
// its own work back rather than applying it on top of the winner's.
func TestInbox_CompleteIsRefusedOnAnAlreadyCompletedRecord(t *testing.T) {
	pool := testDB(t)
	repo := NewInboxRepo(sqlc.New(pool))

	id := insertRecord(t, pool, repo, "test.complete_twice", 300)

	first, err := repo.Complete(context.Background(), id, "owner-a")
	if err != nil {
		t.Fatalf("first Complete: %v", err)
	}
	second, err := repo.Complete(context.Background(), id, "owner-a")
	if err != nil {
		t.Fatalf("second Complete: %v", err)
	}

	if !first {
		t.Error("the first Complete should have matched")
	}
	if second {
		t.Error("the second Complete matched; a duplicate attempt would commit its work")
	}
}

// The design's central claim: two transactions racing the same record serialize on its row lock, and
// exactly one comes away having completed it. The loser must block until the winner commits and then
// see zero rows — not read a stale 'received' and commit alongside it.
func TestInbox_ConcurrentCompletesSerialiseAndExactlyOneWins(t *testing.T) {
	pool := testDB(t)
	id := insertRecord(t, pool, NewInboxRepo(sqlc.New(pool)), "test.complete_concurrent", 300)

	ctx := context.Background()
	txA, err := pool.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin A: %v", err)
	}
	txB, err := pool.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin B: %v", err)
	}

	queries := sqlc.New(pool)
	repoA := NewInboxRepo(queries.WithTx(txA))
	repoB := NewInboxRepo(queries.WithTx(txB))

	// A takes the row lock first and holds it.
	wonA, err := repoA.Complete(ctx, id, "owner-a")
	if err != nil {
		t.Fatalf("A Complete: %v", err)
	}

	// B's UPDATE must not return while A holds the row lock. returned flips only when it does, so a
	// B that read a stale 'received' and sailed past the lock fails the test rather than passing it by
	// happening to report zero rows.
	var wonB bool
	var errB error
	var returned atomic.Bool
	var wg sync.WaitGroup
	wg.Add(1)
	started := make(chan struct{})
	go func() {
		defer wg.Done()
		close(started)
		wonB, errB = repoB.Complete(ctx, id, "owner-a")
		returned.Store(true)
	}()

	<-started
	time.Sleep(300 * time.Millisecond)
	if returned.Load() {
		t.Fatal("B's Complete returned while A still held the row lock; the two attempts did not serialize")
	}

	if err := txA.Commit(); err != nil {
		t.Fatalf("commit A: %v", err)
	}
	wg.Wait()
	if !returned.Load() {
		t.Fatal("B's Complete never returned after A committed")
	}

	if errB != nil {
		t.Fatalf("B Complete: %v", errB)
	}
	if err := txB.Commit(); err != nil {
		t.Fatalf("commit B: %v", err)
	}

	if !wonA {
		t.Error("A should have completed the record")
	}
	if wonB {
		t.Error("B completed the same record: both attempts would have committed their work")
	}
}

// Two consumers reaching the same abandoned record must not both proceed to the handler.
func TestInbox_ConcurrentClaimsYieldExactlyOneWinner(t *testing.T) {
	pool := testDB(t)
	repo := NewInboxRepo(sqlc.New(pool))

	id := insertRecord(t, pool, repo, "test.claim_concurrent", 300)
	if _, err := pool.Exec(
		"UPDATE message_inbox SET lock_expires_at = DATE_SUB(NOW(3), INTERVAL 1 SECOND) WHERE id = ?", id,
	); err != nil {
		t.Fatalf("expiring the lease: %v", err)
	}

	const racers = 8
	results := make([]bool, racers)
	errs := make([]error, racers)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = repo.Claim(context.Background(), id, "owner-racer", 300)
		}(i)
	}
	close(start)
	wg.Wait()

	won := 0
	for i := range results {
		if errs[i] != nil {
			t.Fatalf("racer %d: %v", i, errs[i])
		}
		if results[i] {
			won++
		}
	}
	if won != 1 {
		t.Errorf("%d racers claimed the lease, want exactly 1", won)
	}
}

// MarkFailed has to release the lease, or a failed message waits out its whole lease before anything
// can retry it.
func TestInbox_MarkFailedReleasesTheLease(t *testing.T) {
	pool := testDB(t)
	repo := NewInboxRepo(sqlc.New(pool))

	id := insertRecord(t, pool, repo, "test.mark_failed", 300)

	if err := repo.MarkFailed(context.Background(), id, "owner-a", "handler exploded"); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}

	var status string
	var attempts int
	var lastErr sql.NullString
	var failedAt sql.NullTime
	var owner sql.NullString
	var expires sql.NullTime
	row := pool.QueryRow(
		"SELECT status, attempts, last_error, failed_at, lock_owner, lock_expires_at FROM message_inbox WHERE id = ?", id)
	if err := row.Scan(&status, &attempts, &lastErr, &failedAt, &owner, &expires); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if status != "received" {
		t.Errorf("status = %q, want received so the message stays retryable", status)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
	if lastErr.String != "handler exploded" {
		t.Errorf("last_error = %q", lastErr.String)
	}
	if !failedAt.Valid {
		t.Error("failed_at was not stamped")
	}
	if owner.Valid || expires.Valid {
		t.Error("the lease was not released, so nothing can retry until it lapses")
	}

	// A released lease is immediately claimable, which is what makes the retry possible.
	claimed, err := repo.Claim(context.Background(), id, "owner-b", 300)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if !claimed {
		t.Error("a failed message could not be claimed for retry")
	}
}

func TestInbox_MarkDiscardedIsTerminalAndPurgeable(t *testing.T) {
	pool := testDB(t)
	repo := NewInboxRepo(sqlc.New(pool))

	id := insertRecord(t, pool, repo, "test.discarded", 300)

	if err := repo.MarkDiscarded(context.Background(), id, "owner-a", "no account on event"); err != nil {
		t.Fatalf("MarkDiscarded: %v", err)
	}

	var status string
	var lastErr sql.NullString
	var failedAt, processedAt sql.NullTime
	row := pool.QueryRow(
		"SELECT status, last_error, failed_at, processed_at FROM message_inbox WHERE id = ?", id)
	if err := row.Scan(&status, &lastErr, &failedAt, &processedAt); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if status != "discarded" {
		t.Errorf("status = %q, want discarded", status)
	}
	if lastErr.String != "no account on event" {
		t.Errorf("last_error = %q, want the discard reason", lastErr.String)
	}
	if !failedAt.Valid {
		t.Error("failed_at was not stamped")
	}
	if !processedAt.Valid {
		t.Error("processed_at was not stamped, so the retention purge will never collect the row")
	}

	// Completing a discarded record must be refused, so a late redelivery cannot resurrect it.
	completed, err := repo.Complete(context.Background(), id, "owner-a")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if completed {
		t.Error("a discarded record was completed")
	}
}

// The lease is only worth something if the writes that release it check who holds it. An attempt
// whose own lease lapsed while it was still running must not be able to clear the lease of the
// consumer that legitimately claimed the record after it — doing so lets a third delivery claim a
// record that is actively being worked, and the message is applied twice.

func TestInbox_MarkFailedIsRefusedForAnAttemptThatLostItsLease(t *testing.T) {
	pool := testDB(t)
	repo := NewInboxRepo(sqlc.New(pool))

	id := insertRecord(t, pool, repo, "test.failed_wrong_owner", 300)

	// owner-b takes over after owner-a's lease lapses.
	if _, err := pool.Exec(
		"UPDATE message_inbox SET lock_expires_at = DATE_SUB(NOW(3), INTERVAL 1 SECOND) WHERE id = ?", id,
	); err != nil {
		t.Fatalf("expiring the lease: %v", err)
	}
	claimed, err := repo.Claim(context.Background(), id, "owner-b", 300)
	if err != nil || !claimed {
		t.Fatalf("Claim by owner-b: claimed=%v err=%v", claimed, err)
	}

	// owner-a finally fails, long after it stopped holding the record.
	if err := repo.MarkFailed(context.Background(), id, "owner-a", "late failure"); err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}

	var owner sql.NullString
	var expires sql.NullTime
	var lastError sql.NullString
	row := pool.QueryRow("SELECT lock_owner, lock_expires_at, last_error FROM message_inbox WHERE id = ?", id)
	if err := row.Scan(&owner, &expires, &lastError); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if owner.String != "owner-b" {
		t.Errorf("lock_owner = %q, want owner-b: a lapsed attempt released the live holder's lease", owner.String)
	}
	if !expires.Valid {
		t.Error("lock_expires_at was cleared, leaving the record claimable while owner-b is working it")
	}
	if lastError.Valid {
		t.Errorf("last_error = %q, want none: owner-a's failure is not owner-b's attempt", lastError.String)
	}
}

func TestInbox_CompleteIsRefusedForAnAttemptThatLostItsLease(t *testing.T) {
	pool := testDB(t)
	repo := NewInboxRepo(sqlc.New(pool))

	id := insertRecord(t, pool, repo, "test.complete_wrong_owner", 300)

	completed, err := repo.Complete(context.Background(), id, "owner-b")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if completed {
		t.Error("Complete succeeded for an owner that does not hold the lease; a slow attempt can " +
			"steal the completion from the consumer that replaced it")
	}
}

func TestInbox_MarkDiscardedIsRefusedForAnAttemptThatLostItsLease(t *testing.T) {
	pool := testDB(t)
	repo := NewInboxRepo(sqlc.New(pool))

	id := insertRecord(t, pool, repo, "test.discard_wrong_owner", 300)

	if err := repo.MarkDiscarded(context.Background(), id, "owner-b", "not mine to end"); err != nil {
		t.Fatalf("MarkDiscarded: %v", err)
	}

	var status string
	if err := pool.QueryRow("SELECT status FROM message_inbox WHERE id = ?", id).Scan(&status); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if status != "received" {
		t.Errorf("status = %q, want received: an attempt with no lease ended someone else's message", status)
	}
}

// A message this handler was never going to act on is terminal and purgeable like a discard, but it
// is not a failure — the monitor scans 'discarded' and never 'ignored'.
func TestInbox_MarkIgnoredIsTerminalAndPurgeable(t *testing.T) {
	pool := testDB(t)
	repo := NewInboxRepo(sqlc.New(pool))

	id := insertRecord(t, pool, repo, "test.ignored", 300)

	if err := repo.MarkIgnored(context.Background(), id, "owner-a", "unhandled event type"); err != nil {
		t.Fatalf("MarkIgnored: %v", err)
	}

	var status string
	var processedAt sql.NullTime
	var owner sql.NullString
	row := pool.QueryRow("SELECT status, processed_at, lock_owner FROM message_inbox WHERE id = ?", id)
	if err := row.Scan(&status, &processedAt, &owner); err != nil {
		t.Fatalf("scan: %v", err)
	}

	if status != "ignored" {
		t.Errorf("status = %q, want ignored", status)
	}
	if !processedAt.Valid {
		t.Error("processed_at is unset, so the retention purge and its index will never reach this row")
	}
	if owner.Valid {
		t.Errorf("lock_owner = %q, want released", owner.String)
	}

	// A terminal record is not re-runnable: the next delivery must find nothing to claim.
	claimed, err := repo.Claim(context.Background(), id, "owner-b", 300)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed {
		t.Error("an ignored record was claimed; the handler would run on a message already ended")
	}
}
