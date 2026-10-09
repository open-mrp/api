package repository

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/open-mrp/api/services/platform-service/internal/domain"
	"github.com/open-mrp/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/open-mrp/api/shared/blobstore"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/contracts"
	"github.com/open-mrp/api/shared/db"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/pagination"
	"github.com/open-mrp/api/shared/tracing"
)

var requestLogRepoTracer = tracing.GetTracer("platform-service.request_log_repository")

type requestLogRepoImpl struct {
	db       *sqlc.Queries
	payloads *blobstore.Store
}

// NewRequestLogRepo builds the repository. payloads reads the bodies of logs stored in object storage; nil when no bucket is configured.
func NewRequestLogRepo(db *sqlc.Queries, payloads *blobstore.Store) domain.RequestLogRepo {
	return &requestLogRepoImpl{db: db, payloads: payloads}
}

// validJSON keeps a malformed stored body from reaching the API, which embeds it as raw JSON: one
// truncated body would otherwise make the whole response unparseable. Substitute the size so the log
// still shows something was there.
func validJSON(s *string) *string {
	if s == nil || json.Valid([]byte(*s)) {
		return s
	}
	placeholder := fmt.Sprintf(`{"_invalid_json":true,"_original_size":%d}`, len(*s))
	return &placeholder
}

func (r *requestLogRepoImpl) Create(ctx context.Context, rl *domain.RequestLog) *apierror.APIError {
	ctx, span := requestLogRepoTracer.Start(ctx, "repository.request_log.create")
	defer span.End()

	payloadKey, apiErr := r.storeInlinePayload(ctx, rl)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	statusCode := rl.StatusCode
	if statusCode < 100 || statusCode > 599 {
		statusCode = 500
	}

	// rl.ActorID is the raw actor id the API exposes — the user_id for a user actor, or the api_key type_id for an api_key actor (set from the identity by the gateway). Stored as-is.
	err := r.db.CreateRequestLog(ctx, sqlc.CreateRequestLogParams{
		ID:                   rl.ID,
		Method:               rl.Method,
		Host:                 rl.Host,
		Path:                 rl.Path,
		NormalizedRoute:      rl.NormalizedRoute,
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
		IdentityType:         db.NullStringPtr(rl.IdentityType),
		ApiVersion:           db.NullStringPtr(rl.APIVersion),
		TraceID:              db.NullStringPtr(rl.TraceID),
		PublicEndpoint:       rl.PublicEndpoint,
		Hidden:               rl.Hidden,
		PayloadKey:           db.NullStringPtr(payloadKey),
	})
	// The id is the request's own, so a duplicate is a redelivery of a log already stored.
	if db.IsDuplicateEntry(err) {
		return nil
	}
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to create request log."))
	}

	return nil
}

// storeInlinePayload stores the bodies of a log the gateway could not upload, so they still reach
// object storage rather than the row. The key derives from the log id, so a redelivered log
// rewrites the same object. A failed upload fails the message, which is redelivered. Without a
// payloads bucket, as on a local stack, the bodies are dropped.
func (r *requestLogRepoImpl) storeInlinePayload(ctx context.Context, rl *domain.RequestLog) (*string, *apierror.APIError) {
	if rl.PayloadKey != nil {
		return rl.PayloadKey, nil
	}
	payload := contracts.RequestLogPayload{
		QueryJSON:        nonEmptyStringPtr(rl.QueryJSON),
		RequestBodyJSON:  nonEmptyStringPtr(rl.BodyJSON),
		ResponseBodyJSON: nonEmptyStringPtr(rl.ResponseJSON),
		StackTrace:       nonEmptyStringPtr(rl.StackTrace),
	}
	if payload == (contracts.RequestLogPayload{}) || r.payloads == nil {
		return nil, nil
	}

	key := contracts.RequestLogPayloadKey(rl.ID)
	if apiErr := r.payloads.Put(ctx, key, payload); apiErr != nil {
		return nil, apiErr
	}
	return &key, nil
}

func (r *requestLogRepoImpl) FindByID(ctx context.Context, id, callerAccountID string, includes []string) (*domain.RequestLogRead, *apierror.APIError) {
	ctx, span := requestLogRepoTracer.Start(ctx, "repository.request_log.find_by_id")
	defer span.End()
	if needsEnrichedFindByID(includes) {
		row, err := r.db.FindRequestLogByID(ctx, sqlc.FindRequestLogByIDParams{
			ID:              id,
			CallerAccountID: db.NullString(callerAccountID),
		})
		if apiErr := db.MapSQLError(err); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		read := mapRowToRequestLogRead(&row)
		if apiErr := r.loadPayload(ctx, read, db.StringFromNullString(row.PayloadKey), includes); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		applyRequestedJSONIncludes(read, includes)
		return read, nil
	}

	row, err := r.db.FindRequestLogBaseByID(ctx, sqlc.FindRequestLogBaseByIDParams{
		ID:              id,
		CallerAccountID: db.NullString(callerAccountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	read := mapBaseRowToRequestLogRead(&row)
	if apiErr := r.loadPayload(ctx, read, db.StringFromNullString(row.PayloadKey), includes); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	applyRequestedJSONIncludes(read, includes)
	return read, nil
}

// loadPayload fills a log's body includes from object storage. It reads the object only when one
// of them was asked for, so a log opened without them never touches the bucket. A log with no
// object, or a body the object lacks, reads as {}: a request or response without a body.
func (r *requestLogRepoImpl) loadPayload(ctx context.Context, read *domain.RequestLogRead, key *string, includes []string) *apierror.APIError {
	if !anyIncludeRequested(includes, "query_params", "request_body", "response_body") {
		return nil
	}
	emptyObject := "{}"
	if key == nil {
		read.BodyJSON = &emptyObject
		read.ResponseJSON = &emptyObject
		return nil
	}
	if r.payloads == nil {
		return apierror.NewInternalError(nil, "Request log payload is stored in object storage, but no payloads bucket is configured.")
	}

	var payload contracts.RequestLogPayload
	if apiErr := r.payloads.Get(ctx, *key, &payload); apiErr != nil {
		return apiErr
	}

	read.QueryJSON = validJSON(payload.QueryJSON)
	read.BodyJSON = cmp.Or(validJSON(payload.RequestBodyJSON), &emptyObject)
	read.ResponseJSON = cmp.Or(validJSON(payload.ResponseBodyJSON), &emptyObject)
	return nil
}

func (r *requestLogRepoImpl) List(ctx context.Context, callerAccountID string, filter *domain.ListRequestLogsFilter, includes []string) (*domain.ListRequestLogsResult, *apierror.APIError) {
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

	var cur *pagination.StringCursor
	var cursorDir *pagination.Direction
	dir := pagination.DirectionForward
	if filter.Cursor != nil {
		decoded, err := pagination.DecodeStringCursor(*filter.Cursor)
		if err != nil {
			return nil, apierror.NewValidationErrorWithParam("Invalid pagination cursor.", "cursor")
		}
		cur = &decoded
		cursorDir = &decoded.Direction
		dir = decoded.Direction
	}

	// actor_id stores the raw id the API exposes (user_id for user actors), so the caller's filter.ActorIDs match rl.actor_id directly — no translation.
	rawSQL, args := buildListQuery(mode, dir, callerAccountID, filter, cur, limit+1)

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

	// The list SQL sorts only the (id, occurred_at) id page and joins the wide columns back without an outer ORDER BY (sorting rows that carry the JSON payload columns exhausts MySQL's sort buffer — see buildListQuery), so keyset order is restored here before pagination slices the page.
	sortListResults(results, dir)

	for _, read := range results {
		applyRequestedJSONIncludes(read, includes)
	}

	paged, pageInfo := pagination.BuildPageString(results, limit, cursorDir, requestLogOccurredAt, requestLogID)
	return &domain.ListRequestLogsResult{RequestLogs: paged, PageInfo: pageInfo}, nil
}

// anyIncludeRequested returns true when any of the given keys is present in includes. When includes is nil (no include param), returns false — no enriched data is needed.
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

// needsEnrichedFindByID returns true when the FindByID query must use the full JOINs (because any sub-object include is requested).
func needsEnrichedFindByID(includes []string) bool {
	return anyIncludeRequested(includes, "account", "actor", "actor.role")
}

type queryMode int

const (
	queryModeBase queryMode = iota
	queryModeActor
	queryModeFull
)

// pickQueryMode picks the cheapest list query variant that satisfies the requested includes. queryModeBase has no joins; queryModeActor joins user+api_key only; queryModeFull joins all related tables.
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
	case "agent":
		// actor_id holds the agent definition id; its display name lives in agent-service (a separate datastore), so it cannot be joined here and is resolved by the api-gateway presenter instead.
		return &domain.RequestLogActor{ID: actorID, ActorType: constants.ActorTypeAgent}
	}
	return nil
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

	identType := db.StringFromNullString(row.IdentityType)
	actorID := row.ActorID.String
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
		case "agent":
			// actor_id holds the agent definition id; its display name lives in agent-service (a separate datastore) and is resolved by the api-gateway presenter, so there is nothing to join here.
			rl.Actor = &domain.RequestLogActor{
				ID:        actorID,
				ActorType: constants.ActorTypeAgent,
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

	return rl
}
