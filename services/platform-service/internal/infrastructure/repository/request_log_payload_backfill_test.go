package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db/backfill"
)

func TestRequestLogPayloadBackfill_MovesBodiesThenPointsRows(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()
	mock.MatchExpectationsInOrder(true)

	store := newTestPayloadStore(t)
	queries := sqlc.New(sqlDB)
	b := NewRequestLogPayloadBackfill(queries, queries, store, time.Now())

	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	mock.ExpectQuery("FORCE INDEX \\(request_log_occurred_at_idx\\)").
		WillReturnRows(sqlmock.NewRows([]string{"id", "occurred_at"}).
			AddRow("rq_a", at).
			AddRow("rq_b", at))
	mock.ExpectQuery("SELECT id, query_json, request_body_json").
		WillReturnRows(sqlmock.NewRows([]string{"id", "query_json", "request_body_json", "response_body_json", "stack_trace"}).
			AddRow("rq_a", nil, []byte(`{"name":"widget"}`), []byte(`{"id":"itm_1"}`), nil).
			AddRow("rq_b", nil, []byte(`{}`), []byte(`{}`), nil))
	mock.ExpectExec("UPDATE request_log").
		WithArgs("rq_a").
		WillReturnResult(sqlmock.NewResult(0, 1))

	next, rows, done, err := b.Batch(context.Background(), &backfill.Meter{}, "", 10)
	if err != nil {
		t.Fatalf("Batch: %v", err)
	}
	if done || rows != 1 {
		t.Errorf("done=%v rows=%d, want false, 1", done, rows)
	}
	if want := formatRequestLogCursor(at, "rq_b"); next != want {
		t.Errorf("cursor: got %q want %q", next, want)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	var payload contracts.RequestLogPayload
	if apiErr := store.Get(context.Background(), contracts.RequestLogPayloadKey("rq_a"), &payload); apiErr != nil {
		t.Fatalf("stored payload: %v", apiErr)
	}
	if payload.RequestBodyJSON == nil || *payload.RequestBodyJSON != `{"name":"widget"}` {
		t.Errorf("request body: got %v", payload.RequestBodyJSON)
	}
}

func TestRequestLogPayloadBackfill_EmptyPageFinishes(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	queries := sqlc.New(sqlDB)
	b := NewRequestLogPayloadBackfill(queries, queries, newTestPayloadStore(t), time.Now())
	mock.ExpectQuery("FORCE INDEX").WillReturnRows(sqlmock.NewRows([]string{"id", "occurred_at"}))

	cursor := formatRequestLogCursor(time.Now(), "rq_z")
	next, _, done, err := b.Batch(context.Background(), &backfill.Meter{}, cursor, 10)
	if err != nil || !done || next != cursor {
		t.Fatalf("expected done at %q, got next=%q done=%v err=%v", cursor, next, done, err)
	}
}

func TestRequestLogCursor_RoundTrips(t *testing.T) {
	t.Parallel()
	at := time.Date(2026, 5, 6, 7, 8, 9, 123000000, time.UTC)
	gotAt, gotID, err := parseRequestLogCursor(formatRequestLogCursor(at, "rq_1"))
	if err != nil || !gotAt.Equal(at) || gotID != "rq_1" {
		t.Fatalf("got %v %q %v", gotAt, gotID, err)
	}
	if _, _, err := parseRequestLogCursor("garbage"); err == nil {
		t.Error("a malformed cursor must be rejected")
	}
}

// The update builds the key in SQL; it must name the same object the gateway and reader use.
func TestPointRequestLogsAtPayloads_KeyMatchesContract(t *testing.T) {
	t.Parallel()
	if contracts.RequestLogPayloadKey("ID") != "request-logs/ID.json.gz" {
		t.Fatal("contracts.RequestLogPayloadKey changed; update PointRequestLogsAtPayloads's CONCAT to match")
	}
}
