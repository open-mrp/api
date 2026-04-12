package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/augno/api/services/platform-service/internal/domain"
	"github.com/augno/api/services/platform-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/pagination"
	"github.com/augno/api/shared/tracing"
)

var auditEventRepoTracer = tracing.GetTracer("platform-service.audit_event_repository")

type auditEventRepoImpl struct {
	db *sqlc.Queries
}

func NewAuditEventRepo(db *sqlc.Queries) domain.AuditEventRepo {
	return &auditEventRepoImpl{db: db}
}

func (r *auditEventRepoImpl) Create(ctx context.Context, event *domain.AuditEvent) *apierror.APIError {
	ctx, span := auditEventRepoTracer.Start(ctx, "repository.audit_event.create")
	defer span.End()

	if event == nil {
		return apierror.NewValidationError("audit event is required")
	}

	var changesParam db.NullableRawMessage
	if event.Changes != nil {
		changesJSON, err := json.Marshal(event.Changes)
		if err != nil {
			return tracing.Trace(span, apierror.NewInternalError(err, "Failed to marshal audit event changes."))
		}
		changesParam = db.NullableRawMessage(changesJSON)
	}

	var metadataParam db.NullableRawMessage
	if len(event.Metadata) > 0 {
		metadataParam = db.NullableRawMessage(event.Metadata)
	}

	err := r.db.CreateAuditEvent(ctx, sqlc.CreateAuditEventParams{
		TypeID:           event.ID,
		ActorID:          event.ActorID,
		ActorType:        event.ActorType,
		IdentityType:     event.IdentityType,
		AccountID:        event.AccountID,
		Action:           string(event.Action),
		ResourceType:     string(event.ResourceType),
		ResourceID:       event.ResourceID,
		Changes:          changesParam,
		Metadata:         metadataParam,
		ServiceName:      event.ServiceName,
		RequestID:        db.NullStringPtr(event.RequestID),
		IdempotencyKeyID: db.NullStringPtr(event.IdempotencyKeyID),
		SourceIp:         db.NullStringPtr(event.SourceIP),
		OccurredAt:       event.OccurredAt,
	})
	if err != nil {
		return tracing.Trace(span, apierror.NewInternalError(err, "Failed to create audit event."))
	}

	return nil
}

func (r *auditEventRepoImpl) FindByID(ctx context.Context, id string, targetAccountID string, includes []string) (*domain.AuditEventRead, *apierror.APIError) {
	ctx, span := auditEventRepoTracer.Start(ctx, "repository.audit_event.find_by_id")
	defer span.End()

	row, err := r.db.FindAuditEventByID(ctx, sqlc.FindAuditEventByIDParams{
		IncludeChanges:  includeJSONFieldParam(includes, "changes"),
		IncludeMetadata: includeJSONFieldParam(includes, "metadata"),
		TypeID:          id,
		AccountID:       targetAccountID,
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	read := mapAuditEventRowToRead(&row)
	return read, nil
}

func auditActorDisplayName(userName, apiKeyName sql.NullString) *string {
	if userName.Valid && userName.String != "" {
		s := userName.String
		return &s
	}
	if apiKeyName.Valid && apiKeyName.String != "" {
		s := apiKeyName.String
		return &s
	}
	return nil
}

func auditActorHandle(identityType string, userEmail, apiKeyRedactedValue sql.NullString) *string {
	if identityType == "user" && userEmail.Valid && userEmail.String != "" {
		s := userEmail.String
		return &s
	}
	if identityType == "api_key" && apiKeyRedactedValue.Valid && apiKeyRedactedValue.String != "" {
		s := apiKeyRedactedValue.String
		return &s
	}
	return nil
}

func (r *auditEventRepoImpl) List(ctx context.Context, targetAccountID string, filter *domain.ListAuditEventsFilter, includes []string) (*domain.ListAuditEventsResult, *apierror.APIError) {
	ctx, span := auditEventRepoTracer.Start(ctx, "repository.audit_event.list")
	defer span.End()

	includeChanges := includeJSONFieldParam(includes, "changes")
	includeMetadata := includeJSONFieldParam(includes, "metadata")

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	var cursorDir *pagination.Direction
	var cur *pagination.StringCursor
	if filter.Cursor != nil {
		decoded, err := pagination.DecodeStringCursor(*filter.Cursor)
		if err != nil {
			return nil, apierror.NewValidationError("Invalid pagination cursor.")
		}
		cursorDir = &decoded.Direction
		cur = &decoded
	}

	resourceTypeFilter := buildExactStringFilter(filter.ResourceType)
	resourceIDFilter := buildExactStringFilter(filter.ResourceID)
	actorIDFilter := buildExactStringFilter(filter.ActorID)
	actionFilter := buildExactStringFilter(filter.Action)

	searchQuery := sql.NullString{}
	if filter.Query != nil && *filter.Query != "" {
		searchQuery = sql.NullString{String: "%" + *filter.Query + "%", Valid: true}
	}

	startDate := db.NullTimePtr(filter.StartDate)
	endDate := db.NullTimePtr(filter.EndDate)

	if cursorDir == nil || *cursorDir == pagination.DirectionForward {
		// Forward query order DESC, returned rows are already in correct
		// direction for BuildPageString.
		var cursorOccurredAt sql.NullTime
		var cursorID sql.NullString
		if cur != nil {
			cursorOccurredAt = sql.NullTime{Time: cur.OccurredAt, Valid: true}
			cursorID = sql.NullString{String: cur.ID, Valid: true}
		}

		rows, err := r.db.ListAuditEventsForward(ctx, sqlc.ListAuditEventsForwardParams{
			IncludeChanges:     includeChanges,
			IncludeMetadata:    includeMetadata,
			TargetAccountID:    targetAccountID,
			ResourceTypeFilter: resourceTypeFilter,
			ResourceIDFilter:   resourceIDFilter,
			ActorIDFilter:      actorIDFilter,
			ActionFilter:       actionFilter,
			StartDate:          startDate,
			EndDate:            endDate,
			SearchQuery:        searchQuery,
			CursorOccurredAt:   cursorOccurredAt,
			CursorID:           cursorID,
			Limit:              limit + 1,
		})
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to query audit events."))
		}

		results := make([]*domain.AuditEventRead, len(rows))
		for i := range rows {
			results[i] = mapAuditEventRowToRead(&rows[i])
		}

		paged, pageInfo := pagination.BuildPageString(results, limit, cursorDir, auditEventOccurredAt, auditEventID)
		return &domain.ListAuditEventsResult{AuditEvents: paged, PageInfo: pageInfo}, nil
	}

	// Backward direction: query order ASC and BuildPageString will reverse.
	rows, err := r.db.ListAuditEventsBackward(ctx, sqlc.ListAuditEventsBackwardParams{
		IncludeChanges:     includeChanges,
		IncludeMetadata:    includeMetadata,
		TargetAccountID:    targetAccountID,
		ResourceTypeFilter: resourceTypeFilter,
		ResourceIDFilter:   resourceIDFilter,
		ActorIDFilter:      actorIDFilter,
		ActionFilter:       actionFilter,
		StartDate:          startDate,
		EndDate:            endDate,
		SearchQuery:        searchQuery,
		CursorOccurredAt:   cur.OccurredAt,
		CursorID:           cur.ID,
		Limit:              limit + 1,
	})
	if err != nil {
		return nil, tracing.Trace(span, apierror.NewInternalError(err, "Failed to query audit events."))
	}

	results := make([]*domain.AuditEventRead, len(rows))
	for i := range rows {
		results[i] = mapAuditEventRowToRead(&rows[i])
	}

	paged, pageInfo := pagination.BuildPageString(results, limit, cursorDir, auditEventOccurredAt, auditEventID)
	return &domain.ListAuditEventsResult{AuditEvents: paged, PageInfo: pageInfo}, nil
}

func auditEventOccurredAt(ae *domain.AuditEventRead) time.Time { return ae.OccurredAt }
func auditEventID(ae *domain.AuditEventRead) string            { return ae.ID }

func mapAuditEventRowToRead(row any) *domain.AuditEventRead {
	switch r := row.(type) {
	case *sqlc.FindAuditEventByIDRow:
		return mapAuditEventBaseRow(r.TypeID, r.ActorID, r.ActorType, r.IdentityType, r.AccountID, r.Action, r.ResourceType, r.ResourceID, r.Changes, r.Metadata, r.ServiceName, r.RequestID, r.IdempotencyKeyID, r.SourceIp, r.OccurredAt, r.CreatedAt, r.UserName, r.UserEmail, r.ApiKeyName, r.ApiKeyRedactedValue)
	case *sqlc.ListAuditEventsForwardRow:
		return mapAuditEventBaseRow(r.TypeID, r.ActorID, r.ActorType, r.IdentityType, r.AccountID, r.Action, r.ResourceType, r.ResourceID, r.Changes, r.Metadata, r.ServiceName, r.RequestID, r.IdempotencyKeyID, r.SourceIp, r.OccurredAt, r.CreatedAt, r.UserName, r.UserEmail, r.ApiKeyName, r.ApiKeyRedactedValue)
	case *sqlc.ListAuditEventsBackwardRow:
		return mapAuditEventBaseRow(r.TypeID, r.ActorID, r.ActorType, r.IdentityType, r.AccountID, r.Action, r.ResourceType, r.ResourceID, r.Changes, r.Metadata, r.ServiceName, r.RequestID, r.IdempotencyKeyID, r.SourceIp, r.OccurredAt, r.CreatedAt, r.UserName, r.UserEmail, r.ApiKeyName, r.ApiKeyRedactedValue)
	default:
		// Should never happen
		return &domain.AuditEventRead{}
	}
}

func mapAuditEventBaseRow(
	typeID string,
	actorID string,
	actorType string,
	identityType string,
	accountID string,
	action string,
	resourceType string,
	resourceID string,
	changes interface{},
	metadata interface{},
	serviceName string,
	requestID sql.NullString,
	idempotencyKeyID sql.NullString,
	sourceIP sql.NullString,
	occurredAt time.Time,
	createdAt time.Time,
	userName sql.NullString,
	userEmail sql.NullString,
	apiKeyName sql.NullString,
	apiKeyRedactedValue sql.NullString,
) *domain.AuditEventRead {
	var changesSlice []domain.AuditFieldChange
	if changesBytes := interfaceToBytes(changes); changesBytes != nil {
		if err := json.Unmarshal(changesBytes, &changesSlice); err != nil {
			// Best-effort parsing: keep null changes on error.
			changesSlice = nil
		}
	}

	var metadataRaw json.RawMessage
	if metadataBytes := interfaceToBytes(metadata); metadataBytes != nil {
		metadataRaw = json.RawMessage(metadataBytes)
	}

	actor := &domain.AuditActor{
		ID:           actorID,
		ActorType:    constants.ActorType(identityType),
		Type:         actorType,
		IdentityType: identityType,
		Name:         auditActorDisplayName(userName, apiKeyName),
		Handle:       auditActorHandle(identityType, userEmail, apiKeyRedactedValue),
	}

	return &domain.AuditEventRead{
		AuditEvent: domain.AuditEvent{
			ID: typeID,

			ActorID:      actorID,
			ActorType:    actorType,
			IdentityType: identityType,
			AccountID:    accountID,

			Action:       constants.AuditAction(action),
			ResourceType: constants.ObjectType(resourceType),
			ResourceID:   resourceID,
			Changes:      changesSlice,
			Metadata:     metadataRaw,

			ServiceName:      serviceName,
			RequestID:        db.StringFromNullString(requestID),
			IdempotencyKeyID: db.StringFromNullString(idempotencyKeyID),
			SourceIP:         db.StringFromNullString(sourceIP),

			OccurredAt: occurredAt,
			CreatedAt:  createdAt,
		},
		Actor: actor,
	}
}

func buildExactStringFilter(val *string) string {
	if val == nil || *val == "" {
		return ""
	}
	return *val
}

func interfaceToBytes(v interface{}) []byte {
	switch x := v.(type) {
	case nil:
		return nil
	case []byte:
		if len(x) == 0 {
			return nil
		}
		return x
	case db.NullableRawMessage:
		if len(x) == 0 {
			return nil
		}
		return []byte(x)
	case string:
		if x == "" {
			return nil
		}
		return []byte(x)
	default:
		return nil
	}
}
