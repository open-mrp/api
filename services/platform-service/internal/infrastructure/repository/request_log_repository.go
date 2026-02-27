package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var requestLogRepoTracer = tracing.GetTracer("platform-service.request_log_repository")

type requestLogRepoImpl struct {
	db *sqlc.Queries
}

func NewRequestLogRepo(db *sqlc.Queries) domain.RequestLogRepo {
	return &requestLogRepoImpl{db: db}
}

func (r *requestLogRepoImpl) Create(ctx context.Context, rl *domain.RequestLog) *apierror.APIError {
	ctx, span := requestLogRepoTracer.Start(ctx, "repository.request_log.create")
	defer span.End()

	queryJSON := db.NullableRawMessage("{}")
	if rl.QueryJSON != nil && *rl.QueryJSON != "" {
		queryJSON = db.NullableRawMessage(*rl.QueryJSON)
	}

	bodyJSON := db.NullableRawMessage("{}")
	if rl.BodyJSON != nil && *rl.BodyJSON != "" {
		bodyJSON = db.NullableRawMessage(*rl.BodyJSON)
	}

	responseJSON := db.NullableRawMessage("{}")
	if rl.ResponseJSON != nil && *rl.ResponseJSON != "" {
		responseJSON = db.NullableRawMessage(*rl.ResponseJSON)
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
		IdempotencyKeyID:     db.NullStringPtr(rl.IdempotencyKeyTypeID),
		ActorID:              db.NullStringPtr(rl.ActorID),
		ActorType:            db.NullStringPtr(rl.ActorType),
		InternalErrorMessage: db.NullStringPtr(rl.InternalErrorMessage),
		StackTrace:           db.NullStringPtr(rl.StackTrace),
		IdentityType:         db.NullStringPtr(rl.IdentityType),
		ApiVersion:           db.NullStringPtr(rl.APIVersion),
		TraceID:              db.NullStringPtr(rl.TraceID),
		PublicEndpoint:       rl.PublicEndpoint,
		RequestBodyJson:             bodyJSON,
		ResponseBodyJson:         responseJSON,
	})
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to create request log."))
	}

	return nil
}

func (r *requestLogRepoImpl) FindByID(ctx context.Context, id, targetAccountID string) (*domain.RequestLogRead, *apierror.APIError) {
	ctx, span := requestLogRepoTracer.Start(ctx, "repository.request_log.find_by_id")
	defer span.End()

	row, err := r.db.FindRequestLogByID(ctx, sqlc.FindRequestLogByIDParams{
		ID:              id,
		TargetAccountID: db.NullString(targetAccountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	return mapRowToRequestLogRead(&row), nil
}

func (r *requestLogRepoImpl) List(ctx context.Context, targetAccountID string, filter *domain.ListRequestLogsFilter) (*domain.ListRequestLogsResult, *apierror.APIError) {
	ctx, span := requestLogRepoTracer.Start(ctx, "repository.request_log.list")
	defer span.End()

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	var queryFilter, queryPathFilter string
	var queryErrorMsgFilter sql.NullString
	if filter.Query != nil && *filter.Query != "" {
		queryFilter = *filter.Query
		likeVal := "%" + *filter.Query + "%"
		queryPathFilter = likeVal
		queryErrorMsgFilter = sql.NullString{String: likeVal, Valid: true}
	}
	methodFilter := buildStringFilter(filter.Method, filter.ExactMatch)
	errorCodeFilter := buildStringFilter(filter.ErrorCode, filter.ExactMatch)
	actorNameFilter := buildStringFilter(filter.ActorName, filter.ExactMatch)

	var cursorDir *pagination.Direction

	if filter.Cursor != nil {
		cur, err := pagination.DecodeStringCursor(*filter.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &cur.Direction

		if cur.Direction == pagination.DirectionBackward {
			rows, err := r.db.ListRequestLogsBackward(ctx, sqlc.ListRequestLogsBackwardParams{
				TargetAccountID:     db.NullString(targetAccountID),
				QueryFilter:         queryFilter,
				QueryPathFilter:     queryPathFilter,
				QueryErrorMsgFilter: queryErrorMsgFilter,
				StartDate:           nullTimePtr(filter.StartDate),
				EndDate:          nullTimePtr(filter.EndDate),
				MethodFilter:     methodFilter,
				StatusCode:       nullInt32Ptr(filter.StatusCode),
				ErrorCodeFilter:  nullStringVal(errorCodeFilter),
				AccountIDFilter:  nullStringPtrVal(filter.AccountID),
				ActorIDFilter:    nullStringPtrVal(filter.ActorID),
				ActorTypeFilter:  nullStringPtrVal(filter.ActorType),
				ActorNameFilter:  nullStringVal(actorNameFilter),
				PublicEndpoint:   nullBoolPtr(filter.PublicEndpoint),
				CursorOccurredAt: cur.OccurredAt,
				CursorID:         cur.ID,
				Limit:            limit + 1,
			})
			if err != nil {
				return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to query request logs."))
			}
			results := make([]*domain.RequestLogRead, len(rows))
			for i := range rows {
				results[i] = mapBackwardRowToRequestLogRead(&rows[i])
			}
			paged, pageInfo := pagination.BuildPageString(results, limit, cursorDir, requestLogOccurredAt, requestLogID)
			return &domain.ListRequestLogsResult{RequestLogs: paged, PageInfo: pageInfo}, nil
		}
	}

	// Forward (default: first page or forward cursor)
	rows, err := r.db.ListRequestLogsForward(ctx, sqlc.ListRequestLogsForwardParams{
		TargetAccountID:     db.NullString(targetAccountID),
		QueryFilter:         queryFilter,
		QueryPathFilter:     queryPathFilter,
		QueryErrorMsgFilter: queryErrorMsgFilter,
		StartDate:           nullTimePtr(filter.StartDate),
		EndDate:         nullTimePtr(filter.EndDate),
		MethodFilter:    methodFilter,
		StatusCode:      nullInt32Ptr(filter.StatusCode),
		ErrorCodeFilter: nullStringVal(errorCodeFilter),
		AccountIDFilter: nullStringPtrVal(filter.AccountID),
		ActorIDFilter:   nullStringPtrVal(filter.ActorID),
		ActorTypeFilter: nullStringPtrVal(filter.ActorType),
		ActorNameFilter: nullStringVal(actorNameFilter),
		PublicEndpoint:  nullBoolPtr(filter.PublicEndpoint),
		CursorOccurredAt: func() sql.NullTime {
			if filter.Cursor == nil {
				return sql.NullTime{}
			}
			cur, _ := pagination.DecodeStringCursor(*filter.Cursor)
			return sql.NullTime{Time: cur.OccurredAt, Valid: true}
		}(),
		CursorID: func() sql.NullString {
			if filter.Cursor == nil {
				return sql.NullString{}
			}
			cur, _ := pagination.DecodeStringCursor(*filter.Cursor)
			return sql.NullString{String: cur.ID, Valid: true}
		}(),
		Limit: limit + 1,
	})
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to query request logs."))
	}

	results := make([]*domain.RequestLogRead, len(rows))
	for i := range rows {
		results[i] = mapForwardRowToRequestLogRead(&rows[i])
	}

	paged, pageInfo := pagination.BuildPageString(results, limit, cursorDir, requestLogOccurredAt, requestLogID)
	return &domain.ListRequestLogsResult{RequestLogs: paged, PageInfo: pageInfo}, nil
}

func requestLogOccurredAt(rl *domain.RequestLogRead) time.Time { return rl.OccurredAt }
func requestLogID(rl *domain.RequestLogRead) string            { return rl.ID }

func buildStringFilter(val *string, exactMatch bool) string {
	if val == nil || *val == "" {
		return ""
	}
	if exactMatch {
		return *val
	}
	return "%" + *val + "%"
}

func nullTimePtr(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

func nullInt32Ptr(v *int32) sql.NullInt32 {
	if v == nil {
		return sql.NullInt32{}
	}
	return sql.NullInt32{Int32: *v, Valid: true}
}

func nullStringVal(s string) sql.NullString {
	return sql.NullString{String: s, Valid: true}
}

func nullStringPtrVal(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{String: "", Valid: true}
	}
	return sql.NullString{String: *s, Valid: true}
}

func nullBoolPtr(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}

func mapRowToRequestLogRead(row *sqlc.FindRequestLogByIDRow) *domain.RequestLogRead {
	rl := &domain.RequestLogRead{
		ID:              row.ID,
		Method:          row.Method,
		Host:            row.Host,
		Path:            row.Path,
		NormalizedRoute: row.NormalizedRoute,
		StatusCode:      row.StatusCode,
		LatencyUs:       row.LatencyUs,
		OccurredAt:      row.OccurredAt,
		CreatedAt:       row.CreatedAt,
		APIVersion:      db.StringFromNullString(row.ApiVersion),
		IdentityType:    db.StringFromNullString(row.IdentityType),
		ClientIP:        db.StringFromNullString(row.ClientIpString),
		UserAgent:       db.StringFromNullString(row.UserAgent),
		Referrer:        db.StringFromNullString(row.Referrer),
		ErrorCode:       db.StringFromNullString(row.ErrorCode),
		ErrorMessage:    db.StringFromNullString(row.ErrorMessage),
		AccountID:       db.StringFromNullString(row.TargetAccountID),
		AccountName:     db.StringFromNullString(row.AccountName),
		IdempotencyKey:  db.StringFromNullString(row.IdempotencyKey),
	}

	if row.QueryJson != nil {
		s := string(row.QueryJson)
		rl.QueryJSON = &s
	}

	if row.RequestBodyJson != nil {
		s := string(row.RequestBodyJson)
		rl.BodyJSON = &s
	}

	if row.ResponseBodyJson != nil {
		s := string(row.ResponseBodyJson)
		rl.ResponseJSON = &s
	}

	identType := db.StringFromNullString(row.IdentityType)
	actorID := db.StringFromNullString(row.ActorID)
	if identType != nil && actorID != nil {
		switch *identType {
		case "user":
			rl.Actor = &domain.RequestLogActor{
				ID:           *actorID,
				ObjectType:   constants.ObjectTypeUser,
				Name:         db.StringFromNullString(row.UserName),
				Email:        db.StringFromNullString(row.UserEmail),
				RoleID:       db.StringFromNullString(row.UserRoleID),
				RoleName:     db.StringFromNullString(row.UserRoleName),
				RoleTypeCode: db.StringFromNullString(row.UserRoleTypeCode),
			}
		case "api_key":
			id := *actorID
			if typeID := db.StringFromNullString(row.ApiKeyTypeID); typeID != nil {
				id = *typeID
			}
			rl.Actor = &domain.RequestLogActor{
				ID:            id,
				ObjectType:    constants.ObjectTypeAPIKey,
				Name:          db.StringFromNullString(row.ApiKeyName),
				RedactedValue: db.StringFromNullString(row.ApiKeyRedactedValue),
				RoleID:        db.StringFromNullString(row.ApiKeyRoleID),
				RoleName:      db.StringFromNullString(row.ApiKeyRoleName),
				RoleTypeCode:  db.StringFromNullString(row.ApiKeyRoleTypeCode),
			}
		}
	}

	return rl
}

func mapForwardRowToRequestLogRead(row *sqlc.ListRequestLogsForwardRow) *domain.RequestLogRead {
	return mapRowToRequestLogRead(&sqlc.FindRequestLogByIDRow{
		ID: row.ID, Method: row.Method, Host: row.Host, Path: row.Path,
		NormalizedRoute: row.NormalizedRoute, QueryJson: row.QueryJson,
		StatusCode: row.StatusCode, LatencyUs: row.LatencyUs,
		ApiVersion: row.ApiVersion, ActorID: row.ActorID,
		ActorType: row.ActorType, IdentityType: row.IdentityType,
		ClientIpString: row.ClientIpString, UserAgent: row.UserAgent,
		Referrer: row.Referrer, ErrorCode: row.ErrorCode,
		ErrorMessage: row.ErrorMessage, OccurredAt: row.OccurredAt,
		CreatedAt: row.CreatedAt, IdempotencyKeyID: row.IdempotencyKeyID,
		RequestBodyJson: row.RequestBodyJson, ResponseBodyJson: row.ResponseBodyJson,
		UserEmail: row.UserEmail, UserName: row.UserName,
		ApiKeyTypeID: row.ApiKeyTypeID, ApiKeyRedactedValue: row.ApiKeyRedactedValue,
		ApiKeyName: row.ApiKeyName, UserRoleID: row.UserRoleID,
		UserRoleName: row.UserRoleName, UserRoleTypeCode: row.UserRoleTypeCode,
		ApiKeyRoleID: row.ApiKeyRoleID, ApiKeyRoleName: row.ApiKeyRoleName,
		ApiKeyRoleTypeCode: row.ApiKeyRoleTypeCode,
		TargetAccountID:    row.TargetAccountID,
		AccountName:        row.AccountName, IdempotencyKey: row.IdempotencyKey,
	})
}

func mapBackwardRowToRequestLogRead(row *sqlc.ListRequestLogsBackwardRow) *domain.RequestLogRead {
	return mapRowToRequestLogRead(&sqlc.FindRequestLogByIDRow{
		ID: row.ID, Method: row.Method, Host: row.Host, Path: row.Path,
		NormalizedRoute: row.NormalizedRoute, QueryJson: row.QueryJson,
		StatusCode: row.StatusCode, LatencyUs: row.LatencyUs,
		ApiVersion: row.ApiVersion, ActorID: row.ActorID,
		ActorType: row.ActorType, IdentityType: row.IdentityType,
		ClientIpString: row.ClientIpString, UserAgent: row.UserAgent,
		Referrer: row.Referrer, ErrorCode: row.ErrorCode,
		ErrorMessage: row.ErrorMessage, OccurredAt: row.OccurredAt,
		CreatedAt: row.CreatedAt, IdempotencyKeyID: row.IdempotencyKeyID,
		RequestBodyJson: row.RequestBodyJson, ResponseBodyJson: row.ResponseBodyJson,
		UserEmail: row.UserEmail, UserName: row.UserName,
		ApiKeyTypeID: row.ApiKeyTypeID, ApiKeyRedactedValue: row.ApiKeyRedactedValue,
		ApiKeyName: row.ApiKeyName, UserRoleID: row.UserRoleID,
		UserRoleName: row.UserRoleName, UserRoleTypeCode: row.UserRoleTypeCode,
		ApiKeyRoleID: row.ApiKeyRoleID, ApiKeyRoleName: row.ApiKeyRoleName,
		ApiKeyRoleTypeCode: row.ApiKeyRoleTypeCode,
		TargetAccountID:    row.TargetAccountID,
		AccountName:        row.AccountName, IdempotencyKey: row.IdempotencyKey,
	})
}
