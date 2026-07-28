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

	// event.ActorID is the raw actor id the API exposes — the user_id for a user actor, or the api_key type_id for an api_key actor (set from the identity by the audit consumer). Stored as-is.
	err := r.db.CreateAuditEvent(ctx, sqlc.CreateAuditEventParams{
		TypeID:           event.ID,
		ActorID:          event.ActorID,
		ActorType:        event.ActorType,
		IdentityType:     event.IdentityType,
		AccountID:        event.AccountID,
		TargetAccountID:  db.NullStringPtr(event.TargetAccountID),
		Action:           string(event.Action),
		ResourceType:     string(event.ResourceType),
		ResourceID:       event.ResourceID,
		RootResourceType: db.NullString(string(event.RootResourceType)),
		RootResourceID:   db.NullString(event.RootResourceID),
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

func (r *auditEventRepoImpl) FindByID(ctx context.Context, id string, callerAccountID string, includes []string) (*domain.AuditEventRead, *apierror.APIError) {
	ctx, span := auditEventRepoTracer.Start(ctx, "repository.audit_event.find_by_id")
	defer span.End()

	row, err := r.db.FindAuditEventByID(ctx, sqlc.FindAuditEventByIDParams{
		IncludeChanges:        includeJSONFieldParam(includes, "changes"),
		IncludeMetadata:       includeJSONFieldParam(includes, "metadata"),
		TypeID:                id,
		CallerAccountIDActor:  callerAccountID,
		CallerAccountIDTarget: db.NullString(callerAccountID),
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	read := mapAuditEventRowToRead(&row)
	return read, nil
}

func (r *auditEventRepoImpl) BatchGetResourceCreators(ctx context.Context, callerAccountID, resourceType string, resourceIDs []string) ([]domain.ResourceCreator, *apierror.APIError) {
	ctx, span := auditEventRepoTracer.Start(ctx, "repository.audit_event.batch_get_resource_creators")
	defer span.End()

	if len(resourceIDs) == 0 {
		return nil, nil
	}

	rows, err := r.db.BatchGetResourceCreators(ctx, sqlc.BatchGetResourceCreatorsParams{
		ResourceType:          resourceType,
		ResourceIds:           resourceIDs,
		CallerAccountIDActor:  callerAccountID,
		CallerAccountIDTarget: db.NullString(callerAccountID),
	})
	if err != nil {
		return nil, tracing.Trace(span, db.MapSQLError(err))
	}

	creators := make([]domain.ResourceCreator, 0, len(rows))
	for _, row := range rows {
		creators = append(creators, domain.ResourceCreator{
			ResourceID: row.ResourceID,
			Actor: &domain.AuditActor{
				ID:           row.ActorID,
				ActorType:    constants.ActorType(row.IdentityType),
				Type:         row.ActorType,
				IdentityType: row.IdentityType,
				Name:         auditActorDisplayName(row.UserName, row.ApiKeyName),
				Handle:       auditActorHandle(row.IdentityType, row.UserEmail, row.ApiKeyRedactedValue),
			},
		})
	}
	return creators, nil
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

func (r *auditEventRepoImpl) List(ctx context.Context, callerAccountID string, filter *domain.ListAuditEventsFilter, includes []string) (*domain.ListAuditEventsResult, *apierror.APIError) {
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

	includeResourceTypeFilter := len(filter.ResourceTypes) > 0
	resourceTypes := ensureStringSlice(filter.ResourceTypes)
	includeResourceIDFilter := len(filter.ResourceIDs) > 0
	resourceIDs := ensureStringSlice(filter.ResourceIDs)
	includeActorIDFilter := len(filter.ActorIDs) > 0
	actorIDs := ensureStringSlice(filter.ActorIDs)
	includeActorTypeFilter := len(filter.ActorTypes) > 0
	actorTypes := ensureStringSlice(filter.ActorTypes)
	includeActionFilter := len(filter.Actions) > 0
	actions := ensureStringSlice(filter.Actions)
	includeActorAccountFilter := len(filter.ActorAccountIDs) > 0
	actorAccountIDs := ensureStringSlice(filter.ActorAccountIDs)
	includeTargetAccountFilter := len(filter.TargetAccountIDs) > 0
	targetAccountIDs := ensureNullStringSlice(filter.TargetAccountIDs)
	includeRootFilter := filter.RootResourceType != "" && filter.RootResourceID != ""

	searchQuery := sql.NullString{}
	if filter.Query != nil && *filter.Query != "" {
		searchQuery = sql.NullString{String: "%" + db.EscapeLike(*filter.Query) + "%", Valid: true}
	}

	startDate := db.NullTimePtr(filter.StartDate)
	endDate := db.NullTimePtr(filter.EndDate)

	if cursorDir == nil || *cursorDir == pagination.DirectionForward {
		// Forward query order DESC, returned rows are already in correct direction for BuildPageString.
		var cursorOccurredAt sql.NullTime
		var cursorID sql.NullString
		if cur != nil {
			cursorOccurredAt = sql.NullTime{Time: cur.OccurredAt, Valid: true}
			cursorID = sql.NullString{String: cur.ID, Valid: true}
		}

		rows, err := r.db.ListAuditEventsForward(ctx, sqlc.ListAuditEventsForwardParams{
			IncludeChanges:             includeChanges,
			IncludeMetadata:            includeMetadata,
			CallerAccountIDActor:       callerAccountID,
			CallerAccountIDTarget:      db.NullString(callerAccountID),
			IncludeActorAccountFilter:  includeActorAccountFilter,
			ActorAccountIds:            actorAccountIDs,
			IncludeTargetAccountFilter: includeTargetAccountFilter,
			TargetAccountIds:           targetAccountIDs,
			IncludeResourceTypeFilter:  includeResourceTypeFilter,
			ResourceTypes:              resourceTypes,
			IncludeResourceIDFilter:    includeResourceIDFilter,
			ResourceIds:                resourceIDs,
			IncludeRootFilter:          includeRootFilter,
			RootResourceType:           db.NullString(filter.RootResourceType),
			RootResourceID:             db.NullString(filter.RootResourceID),
			IncludeActorIDFilter:       includeActorIDFilter,
			ActorIds:                   actorIDs,
			IncludeActorTypeFilter:     includeActorTypeFilter,
			ActorTypes:                 actorTypes,
			IncludeActionFilter:        includeActionFilter,
			Actions:                    actions,
			StartDate:                  startDate,
			EndDate:                    endDate,
			SearchQuery:                searchQuery,
			CursorOccurredAt:           cursorOccurredAt,
			CursorID:                   cursorID,
			Limit:                      limit + 1,
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
		IncludeChanges:             includeChanges,
		IncludeMetadata:            includeMetadata,
		CallerAccountIDActor:       callerAccountID,
		CallerAccountIDTarget:      db.NullString(callerAccountID),
		IncludeActorAccountFilter:  includeActorAccountFilter,
		ActorAccountIds:            actorAccountIDs,
		IncludeTargetAccountFilter: includeTargetAccountFilter,
		TargetAccountIds:           targetAccountIDs,
		IncludeResourceTypeFilter:  includeResourceTypeFilter,
		ResourceTypes:              resourceTypes,
		IncludeResourceIDFilter:    includeResourceIDFilter,
		ResourceIds:                resourceIDs,
		IncludeRootFilter:          includeRootFilter,
		RootResourceType:           db.NullString(filter.RootResourceType),
		RootResourceID:             db.NullString(filter.RootResourceID),
		IncludeActorIDFilter:       includeActorIDFilter,
		ActorIds:                   actorIDs,
		IncludeActionFilter:        includeActionFilter,
		Actions:                    actions,
		StartDate:                  startDate,
		EndDate:                    endDate,
		SearchQuery:                searchQuery,
		CursorOccurredAt:           cur.OccurredAt,
		CursorID:                   cur.ID,
		Limit:                      limit + 1,
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
		return mapAuditEventBaseRow(r.TypeID, r.ActorID, r.ActorType, r.IdentityType, r.AccountID, r.TargetAccountID, r.AccountName, r.AccountCreatedAt, r.AccountUpdatedAt, r.Action, r.ResourceType, r.ResourceID, r.Changes, r.Metadata, r.ServiceName, r.RequestID, r.IdempotencyKeyID, r.SourceIp, r.OccurredAt, r.CreatedAt, r.UserName, r.UserEmail, r.ApiKeyName, r.ApiKeyRedactedValue, r.IdempotencyKey)
	case *sqlc.ListAuditEventsForwardRow:
		return mapAuditEventBaseRow(r.TypeID, r.ActorID, r.ActorType, r.IdentityType, r.AccountID, r.TargetAccountID, r.AccountName, r.AccountCreatedAt, r.AccountUpdatedAt, r.Action, r.ResourceType, r.ResourceID, r.Changes, r.Metadata, r.ServiceName, r.RequestID, r.IdempotencyKeyID, r.SourceIp, r.OccurredAt, r.CreatedAt, r.UserName, r.UserEmail, r.ApiKeyName, r.ApiKeyRedactedValue, r.IdempotencyKey)
	case *sqlc.ListAuditEventsBackwardRow:
		return mapAuditEventBaseRow(r.TypeID, r.ActorID, r.ActorType, r.IdentityType, r.AccountID, r.TargetAccountID, r.AccountName, r.AccountCreatedAt, r.AccountUpdatedAt, r.Action, r.ResourceType, r.ResourceID, r.Changes, r.Metadata, r.ServiceName, r.RequestID, r.IdempotencyKeyID, r.SourceIp, r.OccurredAt, r.CreatedAt, r.UserName, r.UserEmail, r.ApiKeyName, r.ApiKeyRedactedValue, r.IdempotencyKey)
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
	targetAccountID sql.NullString,
	accountName sql.NullString,
	accountCreatedAt sql.NullTime,
	accountUpdatedAt sql.NullTime,
	action string,
	resourceType string,
	resourceID string,
	changes any,
	metadata any,
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
	idempotencyKey sql.NullString,
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

	read := &domain.AuditEventRead{
		AuditEvent: domain.AuditEvent{
			ID: typeID,

			ActorID:         actorID,
			ActorType:       actorType,
			IdentityType:    identityType,
			AccountID:       accountID,
			TargetAccountID: db.StringFromNullString(targetAccountID),

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
		Actor:          actor,
		AccountName:    db.StringFromNullString(accountName),
		IdempotencyKey: db.StringFromNullString(idempotencyKey),
	}

	if accountCreatedAt.Valid {
		t := accountCreatedAt.Time
		read.AccountCreatedAt = &t
	}
	if accountUpdatedAt.Valid {
		t := accountUpdatedAt.Time
		read.AccountUpdatedAt = &t
	}

	return read
}

func ensureStringSlice(vals []string) []string {
	if len(vals) == 0 {
		return []string{""}
	}
	return vals
}

// ensureNullStringSlice mirrors ensureStringSlice for the nullable target_account_id IN (...) filter: the slice must always have at least one element so sqlc emits a valid placeholder, even when the filter is disabled (the `include_account_filter = false OR ...` guard short-circuits the IN).
func ensureNullStringSlice(vals []string) []sql.NullString {
	if len(vals) == 0 {
		return []sql.NullString{{}}
	}
	out := make([]sql.NullString, len(vals))
	for i, v := range vals {
		out[i] = sql.NullString{String: v, Valid: true}
	}
	return out
}

func interfaceToBytes(v any) []byte {
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
