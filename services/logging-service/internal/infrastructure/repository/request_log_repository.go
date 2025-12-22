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
	if rl.QueryJSON != nil && *rl.QueryJSON != "" {
		queryJSON = json.RawMessage(*rl.QueryJSON)
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
		AccountID:            db.NullStringPtr(rl.AccountID),
		TargetAccountID:      db.NullStringPtr(rl.TargetAccountID),
		ClientIp:             db.NullString(string(rl.ClientIP)),
		ClientIpString:       db.NullStringPtr(rl.ClientIPString),
		UserAgent:            db.NullStringPtr(rl.UserAgent),
		Referrer:             db.NullStringPtr(rl.Referrer),
		ErrorCode:            db.NullStringPtr(rl.ErrorCode),
		ErrorMessage:         db.NullStringPtr(rl.ErrorMessage),
		OccurredAt:           rl.OccurredAt,
		CreatedAt:            rl.CreatedAt,
		IdempotencyKeyID:     db.NullStringPtr(rl.IdempotencyKeyID),
		ActorID:              db.NullStringPtr(rl.ActorID),
		ActorType:            db.NullStringPtr(rl.ActorType),
		InternalErrorMessage: db.NullStringPtr(rl.InternalErrorMessage),
		StackTrace:           db.NullStringPtr(rl.StackTrace),
		IdentityType:         db.NullStringPtr(rl.IdentityType),
	})
	if err != nil {
		return tracing.Trace(span, contracts.NewInternalError(err, "Failed to create request log."))
	}

	return nil
}
