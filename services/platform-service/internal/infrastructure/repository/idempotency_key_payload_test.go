package repository

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/blobstore"
)

func newTestPayloadStore(t *testing.T) *blobstore.Store {
	t.Helper()
	store, err := blobstore.New(&blobstore.Config{Objects: &memObjects{objects: map[string][]byte{}}, Bucket: "payloads"})
	if err != nil {
		t.Fatalf("blobstore.New: %v", err)
	}
	return store
}

// The gateway's reply waits on SetResponse, so a large body is stored inline first and moved to object storage in the background.
func TestIdempotencyKeyRepo_SetResponse_MovesLargeBody(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	mock.MatchExpectationsInOrder(true)

	store := newTestPayloadStore(t)
	repo := NewIdempotencyKeyRepo(sqlDB, sqlc.New(sqlDB), store)

	body := json.RawMessage(`{"data":"` + strings.Repeat("x", blobstore.InlineLimit) + `"}`)
	mock.ExpectExec("UPDATE idempotency_key\\s+SET response_code").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("SET response_body = NULL, response_body_key").
		WithArgs("idempotency/http/ipke_1.json.gz", "ipke_1").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if apiErr := repo.SetResponse(context.Background(), domain.SetResponseParams{ID: "ipke_1", StatusCode: 200, RecoveryPoint: "finished", Body: body}); apiErr != nil {
		t.Fatalf("SetResponse: %v", apiErr)
	}

	deadline := time.Now().Add(2 * time.Second)
	for mock.ExpectationsWereMet() != nil {
		if time.Now().After(deadline) {
			t.Fatalf("response was not moved: %v", mock.ExpectationsWereMet())
		}
		time.Sleep(5 * time.Millisecond)
	}

	got, apiErr := store.GetJSON(context.Background(), "idempotency/http/ipke_1.json.gz")
	if apiErr != nil || string(got) != string(body) {
		t.Fatalf("stored response does not match the cached body: %v", apiErr)
	}
}

func TestIdempotencyKeyRepo_SetResponse_KeepsSmallBodyInline(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewIdempotencyKeyRepo(sqlDB, sqlc.New(sqlDB), newTestPayloadStore(t))
	mock.ExpectExec("UPDATE idempotency_key").WillReturnResult(sqlmock.NewResult(0, 1))

	if apiErr := repo.SetResponse(context.Background(), domain.SetResponseParams{ID: "ipke_2", StatusCode: 201, RecoveryPoint: "finished", Body: json.RawMessage(`{"id":"un_1"}`)}); apiErr != nil {
		t.Fatalf("SetResponse: %v", apiErr)
	}
	time.Sleep(20 * time.Millisecond)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("a small response must not be moved: %v", err)
	}
}
