package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/augno/api/services/logging-service/internal/domain"
	"github.com/augno/api/services/logging-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/db"

	"github.com/stretchr/testify/require"
)

type recordingDB struct {
	lastQuery string
	lastArgs  []any
	err       error
}

func (m *recordingDB) ExecContext(_ context.Context, query string, args ...any) (sql.Result, error) {
	m.lastQuery = query
	m.lastArgs = args
	return dummyResult{}, m.err
}

func (m *recordingDB) PrepareContext(context.Context, string) (*sql.Stmt, error) {
	return nil, errors.New("not implemented")
}
func (m *recordingDB) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, errors.New("not implemented")
}
func (m *recordingDB) QueryRowContext(context.Context, string, ...any) *sql.Row { return nil }

type dummyResult struct{}

func (dummyResult) LastInsertId() (int64, error) { return 0, nil }
func (dummyResult) RowsAffected() (int64, error) { return 0, nil }

func TestCreate_UsesProvidedValues(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	queryJSON := `{"ok":true}`

	recorder := &recordingDB{}
	repo := NewRequestLogRepo(sqlc.New(recorder))

	rl := &domain.RequestLog{
		ID:                   "req-1",
		Method:               "GET",
		Host:                 "example.com",
		Path:                 "/v1/test",
		NormalizedRoute:      "/v1/test",
		QueryJSON:            queryJSON,
		StatusCode:           201,
		LatencyUs:            12345,
		AccountID:            "acct-1",
		ClientIP:             []byte("10.0.0.1"),
		ClientIPString:       "10.0.0.1",
		UserAgent:            "agent",
		Referrer:             "ref",
		ErrorCode:            "E123",
		ErrorMessage:         "bad",
		OccurredAt:           now,
		IdempotencyKeyID:     "idem-1",
		ActorID:              "actor-1",
		ActorType:            "user",
		InternalErrorMessage: "internal",
		StackTrace:           "trace",
		IdentityType:         "user",
	}

	apiErr := repo.Create(ctx, rl)
	require.Nil(t, apiErr)

	require.True(t, strings.Contains(strings.ToLower(recorder.lastQuery), "insert into request_log"))

	expectedArgs := []any{
		rl.ID,
		rl.Method,
		rl.Host,
		rl.Path,
		rl.NormalizedRoute,
		json.RawMessage(queryJSON),
		rl.StatusCode,
		rl.LatencyUs,
		db.NullString(rl.AccountID),
		db.NullString(string(rl.ClientIP)),
		db.NullString(rl.ClientIPString),
		db.NullString(rl.UserAgent),
		db.NullString(rl.Referrer),
		db.NullString(rl.ErrorCode),
		db.NullString(rl.ErrorMessage),
		rl.OccurredAt,
		db.NullString(rl.IdempotencyKeyID),
		db.NullString(rl.ActorID),
		db.NullString(rl.ActorType),
		db.NullString(rl.InternalErrorMessage),
		db.NullString(rl.StackTrace),
		db.NullString(rl.IdentityType),
	}

	require.Equal(t, expectedArgs, recorder.lastArgs)
}

func TestCreate_FallsBackTo500WhenOutOfRange(t *testing.T) {
	ctx := context.Background()
	recorder := &recordingDB{}
	repo := NewRequestLogRepo(sqlc.New(recorder))

	rl := &domain.RequestLog{
		ID:         "req-2",
		StatusCode: 42, // below valid HTTP range
		OccurredAt: time.Now(),
	}

	apiErr := repo.Create(ctx, rl)
	require.Nil(t, apiErr)

	require.Len(t, recorder.lastArgs, 22)
	require.Equal(t, int32(500), recorder.lastArgs[6])
}

func TestCreate_PropagatesDatabaseError(t *testing.T) {
	ctx := context.Background()
	recorder := &recordingDB{err: errors.New("db failure")}
	repo := NewRequestLogRepo(sqlc.New(recorder))

	rl := &domain.RequestLog{
		ID:         "req-3",
		StatusCode: 200,
		OccurredAt: time.Now(),
	}

	apiErr := repo.Create(ctx, rl)
	require.NotNil(t, apiErr)
	require.Equal(t, contracts.ErrorCodeInternalError, apiErr.Code)
}
