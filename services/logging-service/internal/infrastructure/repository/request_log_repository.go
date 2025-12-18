package repository

import (
	"context"
	"encoding/json"

	"github.com/augno/api/services/logging-service/internal/domain"
	"github.com/augno/api/services/logging-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/db"
	"github.com/augno/api/shared/tracing"
)

var requestLogRepoTracer = tracing.GetTracer("logging-service.request_log_repository")

type requestLogRepoImpl struct {
	db *sqlc.Queries
}

func NewRequestLogRepo(db *sqlc.Queries) domain.RequestLogRepo {
	return &requestLogRepoImpl{db: db}
}

func (r *requestLogRepoImpl) Create(ctx context.Context, rl *domain.RequestLog) *contracts.APIError {
	ctx, span := requestLogRepoTracer.Start(ctx, "repository.request_log.create")
	defer span.End()

	var queryJSON json.RawMessage
	if rl.QueryJSON != "" {
		queryJSON = json.RawMessage(rl.QueryJSON)
	}

	statusCode := rl.StatusCode
	if statusCode < 100 || statusCode > 599 {
		statusCode = 500
	}

	err := r.db.CreateRequestLog(ctx, sqlc.CreateRequestLogParams{
		ID:                   rl.ID,
		Method:               rl.Method,
		Host:                 rl.Host,
		Path:                 rl.Path,
		NormalizedRoute:      rl.NormalizedRoute,
		QueryJson:            queryJSON,
		StatusCode:           statusCode,
		LatencyUs:            rl.LatencyUs,
		AccountID:            db.NullString(rl.AccountID),
		ClientIp:             db.NullString(string(rl.ClientIP)),
		ClientIpString:       db.NullString(rl.ClientIPString),
		UserAgent:            db.NullString(rl.UserAgent),
		Referrer:             db.NullString(rl.Referrer),
		ErrorCode:            db.NullString(rl.ErrorCode),
		ErrorMessage:         db.NullString(rl.ErrorMessage),
		OccurredAt:           rl.OccurredAt,
		IdempotencyKeyID:     db.NullString(rl.IdempotencyKeyID),
		ActorID:              db.NullString(rl.ActorID),
		ActorType:            db.NullString(rl.ActorType),
		InternalErrorMessage: db.NullString(rl.InternalErrorMessage),
		StackTrace:           db.NullString(rl.StackTrace),
		IdentityType:         db.NullString(rl.IdentityType),
	})
	if err != nil {
		return tracing.Trace(span, contracts.NewInternalError(err, "Failed to create request log."))
	}

	return nil
}
