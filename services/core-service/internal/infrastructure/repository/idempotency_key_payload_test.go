package repository

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/open-mrp/api/services/core-service/internal/domain"
	"github.com/open-mrp/api/services/core-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/blobstore"
	"github.com/open-mrp/api/shared/db"
)

func largeResponse() json.RawMessage {
	return json.RawMessage(`{"data":"` + strings.Repeat("x", blobstore.InlineLimit) + `"}`)
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the post-commit move")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// A large response is written inline inside the transaction and moved to object storage only
// after the commit, so caching it never holds the transaction open on an upload.
func TestIdempotencyKeyRepo_SetResponse_MovesLargeBodyAfterCommit(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	mock.MatchExpectationsInOrder(true)

	store, objects := newMemPayloads(t)
	queries := sqlc.New(sqlDB)
	repo := NewIdempotencyKeyRepo(queries, &Payloads{Store: store, Pool: queries})

	body := largeResponse()
	mock.ExpectExec("UPDATE service_idempotency_key\\s+SET response_code").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("SET response_body = NULL, response_body_key").
		WithArgs("idempotency/core-service/siipke_1.json.gz", "siipke_1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	txCtx, scope := db.BeginAfterCommitScope(context.Background())
	if apiErr := repo.SetResponse(txCtx, "siipke_1", 200, body, domain.RecoveryPointFinished); apiErr != nil {
		t.Fatalf("SetResponse: %v", apiErr)
	}
	if len(objects.objects) != 0 {
		t.Fatal("nothing may reach object storage before the transaction commits")
	}

	scope.Committed()
	waitFor(t, func() bool { return mock.ExpectationsWereMet() == nil })

	got, apiErr := store.GetJSON(context.Background(), "idempotency/core-service/siipke_1.json.gz")
	if apiErr != nil {
		t.Fatalf("stored response: %v", apiErr)
	}
	if string(got) != string(body) {
		t.Error("stored response does not match the cached body")
	}
}

func TestIdempotencyKeyRepo_SetResponse_KeepsSmallBodyInline(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	store, objects := newMemPayloads(t)
	queries := sqlc.New(sqlDB)
	repo := NewIdempotencyKeyRepo(queries, &Payloads{Store: store, Pool: queries})

	mock.ExpectExec("UPDATE service_idempotency_key").WillReturnResult(sqlmock.NewResult(0, 1))

	txCtx, scope := db.BeginAfterCommitScope(context.Background())
	if apiErr := repo.SetResponse(txCtx, "siipke_2", 200, json.RawMessage(`{"id":"un_1"}`), domain.RecoveryPointFinished); apiErr != nil {
		t.Fatalf("SetResponse: %v", apiErr)
	}
	scope.Committed()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
	if len(objects.objects) != 0 {
		t.Errorf("a small response must stay inline, found %d objects", len(objects.objects))
	}
}

// A failed upload leaves the row untouched, so a replay still reads the inline body.
func TestIdempotencyKeyRepo_SetResponse_LeavesRowAloneWhenUploadFails(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	failing := &failingObjects{done: make(chan struct{})}
	store, _ := blobstore.New(&blobstore.Config{Objects: failing, Bucket: "payloads"})
	queries := sqlc.New(sqlDB)
	repo := NewIdempotencyKeyRepo(queries, &Payloads{Store: store, Pool: queries})

	mock.ExpectExec("UPDATE service_idempotency_key").WillReturnResult(sqlmock.NewResult(0, 1))

	txCtx, scope := db.BeginAfterCommitScope(context.Background())
	if apiErr := repo.SetResponse(txCtx, "siipke_3", 200, largeResponse(), domain.RecoveryPointFinished); apiErr != nil {
		t.Fatalf("SetResponse: %v", apiErr)
	}
	scope.Committed()

	select {
	case <-failing.done:
	case <-time.After(2 * time.Second):
		t.Fatal("upload was never attempted")
	}
	time.Sleep(20 * time.Millisecond)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("the row must not be repointed after a failed upload: %v", err)
	}
}

func TestIdempotencyKeyRepo_GetByScopeHash_ReadsMovedBody(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	store, _ := newMemPayloads(t)
	body := largeResponse()
	objectKey := responseBodyKey("siipke_4")
	if apiErr := store.PutJSON(context.Background(), objectKey, body); apiErr != nil {
		t.Fatalf("PutJSON: %v", apiErr)
	}
	queries := sqlc.New(sqlDB)
	repo := NewIdempotencyKeyRepo(queries, &Payloads{Store: store, Pool: queries})

	now := time.Now()
	mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{
		"id", "type_id", "service_name", "handler", "idempotency_key", "actor_id", "identity_type",
		"scope_hash", "response_code", "response_body", "recovery_point",
		"locked_at", "lock_owner", "lock_expires_at", "created_at", "updated_at",
		"last_run_at", "expires_at", "response_body_key",
	}).AddRow(
		1, "siipke_4", "core-service", "handler", "key", nil, "user",
		"hash", 200, nil, "finished",
		nil, nil, nil, now, now,
		now, now, objectKey,
	))

	key, apiErr := repo.GetByScopeHash(context.Background(), "hash")
	if apiErr != nil {
		t.Fatalf("GetByScopeHash: %v", apiErr)
	}
	if string(key.ResponseBody) != string(body) {
		t.Error("a replay must read the moved response body")
	}
}
