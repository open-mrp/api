package repository

import (
	"context"
	"database/sql"
	"slices"
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

	var queryJSON db.NullableRawMessage
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
		Referrer:             nullStringPtrEmptyAsNull(rl.Referrer),
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
		RequestBodyJson:      bodyJSON,
		ResponseBodyJson:     responseJSON,
	})
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to create request log."))
	}

	return nil
}

func (r *requestLogRepoImpl) FindByID(ctx context.Context, id, targetAccountID string, includes []string) (*domain.RequestLogRead, *apierror.APIError) {
	ctx, span := requestLogRepoTracer.Start(ctx, "repository.request_log.find_by_id")
	defer span.End()
	includeQueryJSON := includeJSONFieldParam(includes, "query_params")
	includeRequestBody := includeJSONFieldParam(includes, "request_body")
	includeResponseBody := includeJSONFieldParam(includes, "response_body")

	if needsEnrichedFindByID(includes) {
		row, err := r.db.FindRequestLogByID(ctx, sqlc.FindRequestLogByIDParams{
			IncludeQueryJson:        includeQueryJSON,
			IncludeRequestBodyJson:  includeRequestBody,
			IncludeResponseBodyJson: includeResponseBody,
			ID:                      id,
			TargetAccountID:         db.NullString(targetAccountID),
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		read := mapRowToRequestLogRead(&row)
		applyRequestedJSONIncludes(read, includes)
		return read, nil
	}

	row, err := r.db.FindRequestLogBaseByID(ctx, sqlc.FindRequestLogBaseByIDParams{
		IncludeQueryJson:        includeQueryJSON,
		IncludeRequestBodyJson:  includeRequestBody,
		IncludeResponseBodyJson: includeResponseBody,
		ID:                      id,
		TargetAccountID:         db.NullString(targetAccountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	read := mapBaseRowToRequestLogRead(&row)
	applyRequestedJSONIncludes(read, includes)
	return read, nil
}

func (r *requestLogRepoImpl) List(ctx context.Context, targetAccountID string, filter *domain.ListRequestLogsFilter, includes []string) (*domain.ListRequestLogsResult, *apierror.APIError) {
	ctx, span := requestLogRepoTracer.Start(ctx, "repository.request_log.list")
	defer span.End()

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	mode := pickQueryMode(includes, filter)

	includeQueryJSON := anyIncludeRequested(includes, "query_params")
	includeRequestBody := anyIncludeRequested(includes, "request_body")
	includeResponseBody := anyIncludeRequested(includes, "response_body")

	var cur *pagination.StringCursor
	var cursorDir *pagination.Direction
	dir := pagination.DirectionForward
	if filter.Cursor != nil {
		decoded, err := pagination.DecodeStringCursor(*filter.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cur = &decoded
		cursorDir = &decoded.Direction
		dir = decoded.Direction
	}

	resolvedActorIDs, err := r.resolveActorIDFilter(ctx, filter.ActorIDs)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to resolve actor filter."))
	}
	filter.ActorIDs = resolvedActorIDs

	rawSQL, args := buildListQuery(mode, dir, targetAccountID, filter,
		includeQueryJSON, includeRequestBody, includeResponseBody, cur, limit+1)

	rows, err := r.db.DB().QueryContext(ctx, rawSQL, args...)
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to query request logs."))
	}
	defer rows.Close()

	var results []*domain.RequestLogRead
	switch mode {
	case queryModeActor:
		results, err = scanActorListRows(rows)
	case queryModeFull:
		results, err = scanFullListRows(rows)
	default:
		results, err = scanBaseListRows(rows)
	}
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to scan request logs."))
	}

	for _, read := range results {
		applyRequestedJSONIncludes(read, includes)
	}

	paged, pageInfo := pagination.BuildPageString(results, limit, cursorDir, requestLogOccurredAt, requestLogID)
	return &domain.ListRequestLogsResult{RequestLogs: paged, PageInfo: pageInfo}, nil
}

// resolveActorIDFilter translates the externally-exposed actor ids in an actor
// filter to the raw rl.actor_id values stored in request_log, so the list query
// can filter on the bare, indexed rl.actor_id column instead of a non-sargable
// COALESCE(au.id, rl.actor_id) expression.
//
// The API exposes account_user.id as a user actor's id (via the
// COALESCE(au.id, rl.actor_id) projection), but request_log.actor_id stores the
// underlying user_id, so account_user ids are mapped back to user_id here. API
// key actor ids are already stored as-is (rl.actor_id == api_key type_id) and
// pass through unchanged.
//
// The mapping is by account_user.id (a globally-unique primary key), NOT scoped to
// the viewed account: the dashboard actor picker can surface an account_user that
// belongs to a different account than the one whose logs are being viewed (e.g. an
// internal/admin actor, or a team list scoped to another account). Scoping the
// lookup to the viewed account left those ids unresolved, so the raw account_user
// id was passed through and matched no rows.
//
// Translation is attempted for EVERY supplied id, not just ids carrying the acus_
// type prefix: account_user / user ids are not guaranteed to be prefixed type ids
// (UUID-keyed identity records exist), so a prefix gate silently skipped the
// translation and filtered the raw account_user id against rl.actor_id — matching
// nothing. Ids that are not account_user ids (api_key type_ids, or ids already
// equal to a user_id) match no account_user row and pass through unchanged.
func (r *requestLogRepoImpl) resolveActorIDFilter(ctx context.Context, actorIDs []string) ([]string, error) {
	if len(actorIDs) == 0 {
		return actorIDs, nil
	}

	rows, err := r.db.ResolveAccountUserActorIDs(ctx, actorIDs)
	if err != nil {
		return nil, err
	}
	userByAccountUser := make(map[string]string, len(rows))
	for _, row := range rows {
		userByAccountUser[row.ID] = row.UserID
	}

	resolved := make([]string, 0, len(actorIDs))
	for _, a := range actorIDs {
		if userID, ok := userByAccountUser[a]; ok {
			resolved = append(resolved, userID)
			continue
		}
		resolved = append(resolved, a)
	}
	return resolved, nil
}

// anyIncludeRequested returns true when any of the given keys is present in includes.
// When includes is nil (no include param), returns false — no enriched data is needed.
func anyIncludeRequested(includes []string, keys ...string) bool {
	if includes == nil {
		return false
	}
	for _, inc := range includes {
		if slices.Contains(keys, inc) {
			return true
		}
	}
	return false
}

func includeJSONFieldParam(includes []string, key string) db.NullableRawMessage {
	if includes == nil {
		return nil
	}
	if slices.Contains(includes, key) {
		return db.NullableRawMessage("1")
	}
	return nil
}

func applyRequestedJSONIncludes(rl *domain.RequestLogRead, includes []string) {
	if rl == nil {
		return
	}
	if !anyIncludeRequested(includes, "query_params") {
		rl.QueryJSON = nil
	}
	if !anyIncludeRequested(includes, "request_body") {
		rl.BodyJSON = nil
	}
	if !anyIncludeRequested(includes, "response_body") {
		rl.ResponseJSON = nil
	}
}

// needsEnrichedFindByID returns true when the FindByID query must use the full JOINs
// (because any sub-object include is requested).
func needsEnrichedFindByID(includes []string) bool {
	return anyIncludeRequested(includes, "account", "actor", "actor.role")
}

type queryMode int

const (
	queryModeBase queryMode = iota
	queryModeActor
	queryModeFull
)

// pickQueryMode picks the cheapest list query variant that satisfies the requested
// includes. queryModeBase has no joins; queryModeActor joins user+api_key
// only; queryModeFull joins all related tables.
func pickQueryMode(includes []string, _ *domain.ListRequestLogsFilter) queryMode {
	if anyIncludeRequested(includes, "account", "actor.role") {
		return queryModeFull
	}
	if anyIncludeRequested(includes, "actor") {
		return queryModeActor
	}
	return queryModeBase
}

func requestLogOccurredAt(rl *domain.RequestLogRead) time.Time { return rl.OccurredAt }
func requestLogID(rl *domain.RequestLogRead) string            { return rl.ID }

func nullStringPtrEmptyAsNull(s *string) sql.NullString {
	if s == nil || *s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

func buildMinimalActor(actorID string, identityType sql.NullString) *domain.RequestLogActor {
	identType := db.StringFromNullString(identityType)
	if actorID == "" || identType == nil {
		return nil
	}
	switch *identType {
	case "user":
		return &domain.RequestLogActor{ID: actorID, ActorType: constants.ActorTypeUser}
	case "api_key":
		return &domain.RequestLogActor{ID: actorID, ActorType: constants.ActorTypeAPIKey}
	}
	return nil
}

func anyToStringPtr(v any) *string {
	switch x := v.(type) {
	case nil:
		return nil
	case string:
		if x == "" {
			return nil
		}
		s := x
		return &s
	case []byte:
		if len(x) == 0 {
			return nil
		}
		s := string(x)
		return &s
	case db.NullableRawMessage:
		if len(x) == 0 {
			return nil
		}
		s := string(x)
		return &s
	default:
		return nil
	}
}

func nonEmptyStringPtr(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
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
		Referrer:        nonEmptyStringPtr(db.StringFromNullString(row.Referrer)),
		ErrorCode:       db.StringFromNullString(row.ErrorCode),
		ErrorMessage:    db.StringFromNullString(row.ErrorMessage),
		AccountID:       db.StringFromNullString(row.TargetAccountID),
		AccountName:     db.StringFromNullString(row.AccountName),
		IdempotencyKey:  db.StringFromNullString(row.IdempotencyKey),
	}

	if row.AccountCreatedAt.Valid {
		t := row.AccountCreatedAt.Time
		rl.AccountCreatedAt = &t
	}
	if row.AccountUpdatedAt.Valid {
		t := row.AccountUpdatedAt.Time
		rl.AccountUpdatedAt = &t
	}

	rl.QueryJSON = anyToStringPtr(row.QueryJson)
	rl.BodyJSON = anyToStringPtr(row.RequestBodyJson)
	rl.ResponseJSON = anyToStringPtr(row.ResponseBodyJson)

	identType := db.StringFromNullString(row.IdentityType)
	actorID := row.ActorID
	if identType != nil && actorID != "" {
		switch *identType {
		case "user":
			rl.Actor = &domain.RequestLogActor{
				ID:        actorID,
				ActorType: constants.ActorTypeUser,
				Name:      db.StringFromNullString(row.UserName),
				Email:     db.StringFromNullString(row.UserEmail),
				RoleID:    db.StringFromNullString(row.UserRoleID),
				RoleName:  db.StringFromNullString(row.UserRoleName),
				RoleType:  db.StringFromNullString(row.UserRoleTypeCode),
			}
		case "api_key":
			id := actorID
			if typeID := db.StringFromNullString(row.ApiKeyTypeID); typeID != nil {
				id = *typeID
			}
			rl.Actor = &domain.RequestLogActor{
				ID:            id,
				ActorType:     constants.ActorTypeAPIKey,
				Name:          db.StringFromNullString(row.ApiKeyName),
				RedactedValue: db.StringFromNullString(row.ApiKeyRedactedValue),
				RoleID:        db.StringFromNullString(row.ApiKeyRoleID),
				RoleName:      db.StringFromNullString(row.ApiKeyRoleName),
				RoleType:      db.StringFromNullString(row.ApiKeyRoleTypeCode),
			}
		}
	}

	return rl
}

func mapBaseRowToRequestLogRead(row *sqlc.FindRequestLogBaseByIDRow) *domain.RequestLogRead {
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
		Referrer:        nonEmptyStringPtr(db.StringFromNullString(row.Referrer)),
		ErrorCode:       db.StringFromNullString(row.ErrorCode),
		ErrorMessage:    db.StringFromNullString(row.ErrorMessage),
		AccountID:       db.StringFromNullString(row.TargetAccountID),
		IdempotencyKey:  db.StringFromNullString(row.IdempotencyKey),
	}

	rl.QueryJSON = anyToStringPtr(row.QueryJson)
	rl.BodyJSON = anyToStringPtr(row.RequestBodyJson)
	rl.ResponseJSON = anyToStringPtr(row.ResponseBodyJson)

	return rl
}
