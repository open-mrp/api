package repository

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/blobstore"
	"github.com/open-mrp/api/shared/contracts"
)

func anyArgs(n int) []driver.Value {
	args := make([]driver.Value, n)
	for i := range args {
		args[i] = sqlmock.AnyArg()
	}
	return args
}

// A log whose gateway upload failed still keeps its bodies out of the row: the consumer stores them.
func TestCreate_StoresInlineBodiesInObjectStorage(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	objects := &memObjects{objects: map[string][]byte{}}
	store, _ := blobstore.New(&blobstore.Config{Objects: objects, Bucket: "payloads"})
	repo := NewRequestLogRepo(sqlc.New(sqlDB), store)

	mock.ExpectExec("INSERT INTO request_log").
		WithArgs(append(anyArgs(26), "request-logs/rq_1.json.gz")...).
		WillReturnResult(sqlmock.NewResult(0, 1))

	body := `{"name":"widget"}`
	if apiErr := repo.Create(context.Background(), &domain.RequestLog{ID: "rq_1", StatusCode: 200, BodyJSON: &body}); apiErr != nil {
		t.Fatalf("Create: %v", apiErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}

	var payload contracts.RequestLogPayload
	if apiErr := store.Get(context.Background(), "request-logs/rq_1.json.gz", &payload); apiErr != nil {
		t.Fatalf("stored payload: %v", apiErr)
	}
	if payload.RequestBodyJSON == nil || *payload.RequestBodyJSON != body {
		t.Errorf("request body: got %v", payload.RequestBodyJSON)
	}
}

// Without a bucket the bodies have nowhere to go; the log row is still written.
func TestCreate_WithoutStoreDropsBodies(t *testing.T) {
	t.Parallel()

	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer sqlDB.Close()

	repo := NewRequestLogRepo(sqlc.New(sqlDB), nil)
	mock.ExpectExec("INSERT INTO request_log").
		WithArgs(append(anyArgs(26), nil)...).
		WillReturnResult(sqlmock.NewResult(0, 1))

	body := `{"name":"widget"}`
	if apiErr := repo.Create(context.Background(), &domain.RequestLog{ID: "rq_2", StatusCode: 200, BodyJSON: &body}); apiErr != nil {
		t.Fatalf("Create: %v", apiErr)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
