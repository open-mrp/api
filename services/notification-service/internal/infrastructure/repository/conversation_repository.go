package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/infrastructure/sqlc"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/tracing"
)

var conversationRepoTracer = tracing.GetTracer("notification-service.conversation_repository")

type conversationRepoImpl struct {
	db *sqlc.Queries
}

func NewConversationRepo(db *sqlc.Queries) domain.ConversationRepo {
	return &conversationRepoImpl{db: db}
}

func (r *conversationRepoImpl) Create(ctx context.Context, id string, input *domain.CreateConversationInput, accountID string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.create")
	defer span.End()

	audience := input.Audience
	if audience == "" {
		audience = "internal"
	}
	err := r.db.CreateConversation(ctx, sqlc.CreateConversationParams{
		ID:                id,
		AccountID:         accountID,
		Type:              input.Type,
		Audience:          audience,
		Title:             db.NullStringPtr(input.Title),
		GroupID:           db.NullStringPtr(input.GroupID),
		TopicResourceType: db.NullStringPtr(input.TopicResourceType),
		TopicResourceID:   db.NullStringPtr(input.TopicResourceID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *conversationRepoImpl) GetByID(ctx context.Context, id, accountID string) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.get_by_id")
	defer span.End()

	row, err := r.db.GetConversationByID(ctx, sqlc.GetConversationByIDParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return conversationFromRow(row), nil
}

func (r *conversationRepoImpl) ListForUser(ctx context.Context, filter domain.ConversationListFilter) ([]*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.list_for_user")
	defer span.End()

	rows, err := r.db.ListConversationsForUser(ctx, sqlc.ListConversationsForUserParams{
		AccountUserID:       db.NullStringPtr(&filter.AccountUserID),
		AccountID:           filter.AccountID,
		Type:                db.NullStringPtr(filter.Type),
		Status:              sql.NullString{String: filter.Status, Valid: filter.Status != ""},
		CursorLastMessageAt: db.NullTimePtr(filter.CursorLastMessageAt),
		CursorID:            db.NullStringPtr(filter.CursorID),
		Limit:               filter.Limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	conversations := make([]*domain.Conversation, 0, len(rows))
	for _, row := range rows {
		c := &domain.Conversation{
			ID:                   row.ID,
			AccountID:            row.AccountID,
			Type:                 row.Type,
			Audience:             row.Audience,
			Title:                db.StringFromNullString(row.Title),
			GroupID:              db.StringFromNullString(row.GroupID),
			TopicResourceType:    db.StringFromNullString(row.TopicResourceType),
			TopicResourceID:      db.StringFromNullString(row.TopicResourceID),
			NextSequence:         row.NextSequence,
			LastMessageID:        db.StringFromNullString(row.LastMessageID),
			LastMessageAt:        db.TimeFromNullTime(row.LastMessageAt),
			IsArchived:           row.IsArchived,
			WorkflowStatus:       db.StringFromNullString(row.WorkflowStatus),
			AssigneeResourceType: db.StringFromNullString(row.AssigneeResourceType),
			AssigneeResourceID:   db.StringFromNullString(row.AssigneeResourceID),
			Metadata:             json.RawMessage(row.Metadata),
			CreatedAt:            row.CreatedAt,
			UpdatedAt:            row.UpdatedAt,
			Hidden:               row.ParticipantHiddenAt.Valid,
		}
		// Unread = messages after the caller's read cursor (their own sends advance it).
		if maxSeq := row.NextSequence - 1; maxSeq > row.ParticipantLastReadSequence {
			c.Unread = maxSeq - row.ParticipantLastReadSequence
		}
		conversations = append(conversations, c)
	}
	return conversations, nil
}

func (r *conversationRepoImpl) LockSequence(ctx context.Context, id, accountID string) (int64, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.lock_sequence")
	defer span.End()

	seq, err := r.db.LockConversationSequence(ctx, sqlc.LockConversationSequenceParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return 0, apiErr
		}
		return 0, tracing.Trace(span, apiErr)
	}
	return seq, nil
}

func (r *conversationRepoImpl) AdvanceAfterMessage(ctx context.Context, id, lastMessageID string, lastMessageAt time.Time) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.advance_after_message")
	defer span.End()

	err := r.db.AdvanceConversationAfterMessage(ctx, sqlc.AdvanceConversationAfterMessageParams{
		LastMessageID: db.NullStringPtr(&lastMessageID),
		LastMessageAt: db.NullTimePtr(&lastMessageAt),
		ID:            id,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *conversationRepoImpl) BindInbox(ctx context.Context, id, accountID, inboxID, externalAddress string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.bind_inbox")
	defer span.End()

	err := r.db.BindConversationInbox(ctx, sqlc.BindConversationInboxParams{
		EmailInboxID:         db.NullString(inboxID),
		EmailExternalAddress: db.NullString(externalAddress),
		ID:                   id,
		AccountID:            accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *conversationRepoImpl) GetDMConversationID(ctx context.Context, accountID, dmKey string) (string, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.get_dm_conversation_id")
	defer span.End()

	id, err := r.db.GetDMConversationID(ctx, sqlc.GetDMConversationIDParams{AccountID: accountID, DmKey: dmKey})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return "", apiErr
		}
		return "", tracing.Trace(span, apiErr)
	}
	return id, nil
}

func (r *conversationRepoImpl) CreateDMKey(ctx context.Context, accountID, dmKey, conversationID string) error {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.create_dm_key")
	defer span.End()
	return r.db.CreateDMKey(ctx, sqlc.CreateDMKeyParams{AccountID: accountID, DmKey: dmKey, ConversationID: conversationID})
}

func (r *conversationRepoImpl) Update(ctx context.Context, id, accountID string, title *string, isArchived *bool, clearTitle bool) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.update")
	defer span.End()

	var archived sql.NullBool
	if isArchived != nil {
		archived = sql.NullBool{Bool: *isArchived, Valid: true}
	}
	// Treat a non-nil pointer (including &"") as a real set so {"title":""} sets an empty string rather than falling back to the old title via COALESCE. db.NullStringPtr would collapse "" to Valid:false and conflate it with 'absent'. The null-clear path is carried separately by clearTitle.
	var titleArg sql.NullString
	if title != nil {
		titleArg = sql.NullString{String: *title, Valid: true}
	}
	err := r.db.UpdateConversation(ctx, sqlc.UpdateConversationParams{
		Title:      titleArg,
		ClearTitle: clearTitle,
		IsArchived: archived,
		ID:         id,
		AccountID:  accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *conversationRepoImpl) SetLegalHold(ctx context.Context, id, accountID string, hold bool) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.set_legal_hold")
	defer span.End()

	err := r.db.SetConversationLegalHold(ctx, sqlc.SetConversationLegalHoldParams{
		LegalHold: hold,
		ID:        id,
		AccountID: accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *conversationRepoImpl) RedactMessages(ctx context.Context, conversationID string) (int64, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.redact_messages")
	defer span.End()

	count, err := r.db.RedactConversationMessages(ctx, conversationID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return count, nil
}

func (r *conversationRepoImpl) GetCustomerSupport(ctx context.Context, vendorAccountID, customerAccountID string) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.get_customer_support")
	defer span.End()

	row, err := r.db.GetCustomerSupportConversation(ctx, sqlc.GetCustomerSupportConversationParams{
		AccountID:         vendorAccountID,
		RelationAccountID: db.NullString(customerAccountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return conversationFromRow(row), nil
}

func (r *conversationRepoImpl) CreateCustomerSupport(ctx context.Context, id, vendorAccountID, customerAccountID string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.create_customer_support")
	defer span.End()

	err := r.db.CreateCustomerSupportConversation(ctx, sqlc.CreateCustomerSupportConversationParams{
		ID:              id,
		AccountID:       vendorAccountID,
		TopicResourceID: db.NullString(customerAccountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *conversationRepoImpl) PromoteToCustomerCase(ctx context.Context, id, accountID string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.promote_to_customer_case")
	defer span.End()

	err := r.db.SetConversationAudienceCustomer(ctx, sqlc.SetConversationAudienceCustomerParams{ID: id, AccountID: accountID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *conversationRepoImpl) SetWorkflowStatus(ctx context.Context, id, accountID, status string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.set_workflow_status")
	defer span.End()

	err := r.db.UpdateConversationWorkflowStatus(ctx, sqlc.UpdateConversationWorkflowStatusParams{
		WorkflowStatus: db.NullString(status),
		ID:             id,
		AccountID:      accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *conversationRepoImpl) Assign(ctx context.Context, id, accountID string, assigneeResourceType, assigneeResourceID *string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.assign")
	defer span.End()

	err := r.db.AssignConversation(ctx, sqlc.AssignConversationParams{
		AssigneeResourceType: db.NullStringPtr(assigneeResourceType),
		AssigneeResourceID:   db.NullStringPtr(assigneeResourceID),
		ID:                   id,
		AccountID:            accountID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *conversationRepoImpl) ListInbox(ctx context.Context, filter domain.SupportInboxFilter) ([]*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.list_inbox")
	defer span.End()

	// Unassigned is a presence-only narg (used solely in an IS NULL guard); pass any non-nil value to enable the "no assignee" filter, nil to disable it.
	var unassigned any
	if filter.Unassigned {
		unassigned = true
	}
	// Hide resolved cases from the default triage view, but keep them visible when the caller filters to a
	// specific status (e.g. the "Resolved" lane) or opens the archived view.
	hideResolved := filter.WorkflowStatus == nil && !filter.IncludeArchived
	rows, err := r.db.ListSupportInbox(ctx, sqlc.ListSupportInboxParams{
		AccountID:           filter.AccountID,
		IsArchived:          filter.IncludeArchived,
		WorkflowStatus:      db.NullStringPtr(filter.WorkflowStatus),
		HideResolved:        hideResolved,
		AssigneeResourceID:  db.NullStringPtr(filter.AssigneeResourceID),
		Unassigned:          unassigned,
		CursorLastMessageAt: db.NullTimePtr(filter.CursorLastMessageAt),
		CursorID:            db.NullStringPtr(filter.CursorID),
		Limit:               filter.Limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.Conversation, 0, len(rows))
	for _, row := range rows {
		out = append(out, conversationFromRow(row))
	}
	return out, nil
}

func (r *conversationRepoImpl) ListByResource(ctx context.Context, accountID, resourceType, resourceID string, limit int32) ([]*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.conversation.list_by_resource")
	defer span.End()

	rows, err := r.db.ListConversationsByResource(ctx, sqlc.ListConversationsByResourceParams{
		AccountID:         accountID,
		TopicResourceType: db.NullString(resourceType),
		TopicResourceID:   db.NullString(resourceID),
		LinkResourceType:  resourceType,
		LinkResourceID:    resourceID,
		Limit:             limit,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	out := make([]*domain.Conversation, 0, len(rows))
	for _, row := range rows {
		out = append(out, conversationFromRow(row))
	}
	return out, nil
}

// ── Participant repo ────────────────────────────────────────────────

type participantRepoImpl struct {
	db *sqlc.Queries
}

func NewParticipantRepo(db *sqlc.Queries) domain.ParticipantRepo {
	return &participantRepoImpl{db: db}
}

func (r *participantRepoImpl) Create(ctx context.Context, p *domain.ConversationParticipant) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.create")
	defer span.End()

	role := p.Role
	if role == "" {
		role = "member"
	}
	err := r.db.CreateParticipant(ctx, sqlc.CreateParticipantParams{
		ID:              p.ID,
		ConversationID:  p.ConversationID,
		AccountID:       p.AccountID,
		ParticipantType: p.ParticipantType,
		AccountUserID:   db.NullStringPtr(p.AccountUserID),
		AgentConfigID:   db.NullStringPtr(p.AgentConfigID),
		Role:            role,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *participantRepoImpl) CreateAgent(ctx context.Context, id, accountID string, input *domain.AddAgentParticipantInput) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.create_agent")
	defer span.End()
	err := r.db.CreateAgentParticipant(ctx, sqlc.CreateAgentParticipantParams{
		ID:                   id,
		ConversationID:       input.ConversationID,
		AccountID:            accountID,
		AgentConfigID:        db.NullString(input.AgentConfigID),
		AgentTriggerPolicy:   db.NullString(input.TriggerPolicy),
		AgentTriggerKeywords: encodeKeywords(input.TriggerKeywords),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *participantRepoImpl) GetByAgentConfigID(ctx context.Context, conversationID, agentConfigID string) (*domain.ConversationParticipant, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.get_by_agent_config_id")
	defer span.End()
	row, err := r.db.GetParticipantByAgentConfigID(ctx, sqlc.GetParticipantByAgentConfigIDParams{
		ConversationID: conversationID,
		AgentConfigID:  db.NullString(agentConfigID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return participantFromRow(row), nil
}

func (r *participantRepoImpl) ReactivateAgent(ctx context.Context, participantID string, input *domain.AddAgentParticipantInput) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.reactivate_agent")
	defer span.End()
	err := r.db.ReactivateAgentParticipant(ctx, sqlc.ReactivateAgentParticipantParams{
		AgentTriggerPolicy:   db.NullString(input.TriggerPolicy),
		AgentTriggerKeywords: encodeKeywords(input.TriggerKeywords),
		ID:                   participantID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *participantRepoImpl) SetStateByID(ctx context.Context, participantID, conversationID, state string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.set_state_by_id")
	defer span.End()
	err := r.db.SetParticipantMembershipByID(ctx, sqlc.SetParticipantMembershipByIDParams{
		Membership:     state,
		ID:             participantID,
		ConversationID: conversationID,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *participantRepoImpl) CreateCustomer(ctx context.Context, id, conversationID, vendorAccountID, customerAccountID, accountUserID string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.create_customer")
	defer span.End()
	err := r.db.CreateCustomerParticipant(ctx, sqlc.CreateCustomerParticipantParams{
		ID:                id,
		ConversationID:    conversationID,
		AccountID:         vendorAccountID,
		RelationAccountID: db.NullString(customerAccountID),
		AccountUserID:     db.NullString(accountUserID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *participantRepoImpl) GetByRelationAccount(ctx context.Context, conversationID, customerAccountID string) (*domain.ConversationParticipant, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.get_by_relation_account")
	defer span.End()
	row, err := r.db.GetParticipantByRelationAccount(ctx, sqlc.GetParticipantByRelationAccountParams{
		ConversationID:    conversationID,
		RelationAccountID: db.NullString(customerAccountID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return participantFromRow(row), nil
}

func (r *participantRepoImpl) AdvanceReadCursorByID(ctx context.Context, participantID, lastReadMessageID string, sequence int64) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.advance_read_cursor_by_id")
	defer span.End()
	err := r.db.AdvanceReadCursorByID(ctx, sqlc.AdvanceReadCursorByIDParams{
		LastReadSequence:   sequence,
		LastReadMessageID:  db.NullString(lastReadMessageID),
		ID:                 participantID,
		LastReadSequence_2: sequence,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// encodeKeywords serializes trigger keywords to a JSON array (nil/empty → NULL).
func encodeKeywords(keywords []string) db.NullableRawMessage {
	if len(keywords) == 0 {
		return nil
	}
	raw, err := json.Marshal(keywords)
	if err != nil {
		return nil
	}
	return db.NullableRawMessage(raw)
}

func (r *participantRepoImpl) Get(ctx context.Context, conversationID, accountUserID string) (*domain.ConversationParticipant, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.get")
	defer span.End()

	row, err := r.db.GetParticipant(ctx, sqlc.GetParticipantParams{
		ConversationID: conversationID,
		AccountUserID:  db.NullStringPtr(&accountUserID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return participantFromRow(row), nil
}

func (r *participantRepoImpl) List(ctx context.Context, conversationID string) ([]*domain.ConversationParticipant, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.list")
	defer span.End()

	rows, err := r.db.ListParticipants(ctx, conversationID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	participants := make([]*domain.ConversationParticipant, 0, len(rows))
	for _, row := range rows {
		participants = append(participants, participantFromRow(row))
	}
	return participants, nil
}

func (r *participantRepoImpl) ListAll(ctx context.Context, conversationID string) ([]*domain.ConversationParticipant, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.list_all")
	defer span.End()

	rows, err := r.db.ListAllParticipants(ctx, conversationID)
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	participants := make([]*domain.ConversationParticipant, 0, len(rows))
	for _, row := range rows {
		participants = append(participants, participantFromRow(row))
	}
	return participants, nil
}

func (r *participantRepoImpl) AdvanceReadCursor(ctx context.Context, conversationID, accountUserID, lastReadMessageID string, sequence int64) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.advance_read_cursor")
	defer span.End()

	err := r.db.AdvanceReadCursor(ctx, sqlc.AdvanceReadCursorParams{
		LastReadSequence:   sequence,
		LastReadMessageID:  db.NullStringPtr(&lastReadMessageID),
		ConversationID:     conversationID,
		AccountUserID:      db.NullStringPtr(&accountUserID),
		LastReadSequence_2: sequence,
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *participantRepoImpl) GetByID(ctx context.Context, participantID, conversationID string) (*domain.ConversationParticipant, *apierror.APIError) {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.get_by_id")
	defer span.End()

	row, err := r.db.GetParticipantByID(ctx, sqlc.GetParticipantByIDParams{ID: participantID, ConversationID: conversationID})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		return nil, tracing.Trace(span, apiErr)
	}
	return participantFromRow(row), nil
}

func (r *participantRepoImpl) SetRole(ctx context.Context, conversationID, accountUserID, role string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.set_role")
	defer span.End()
	err := r.db.SetParticipantRole(ctx, sqlc.SetParticipantRoleParams{
		Role: role, ConversationID: conversationID, AccountUserID: db.NullStringPtr(&accountUserID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *participantRepoImpl) SetState(ctx context.Context, conversationID, accountUserID, state string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.set_state")
	defer span.End()
	err := r.db.SetParticipantMembership(ctx, sqlc.SetParticipantMembershipParams{
		Membership: state, ConversationID: conversationID, AccountUserID: db.NullStringPtr(&accountUserID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *participantRepoImpl) Leave(ctx context.Context, conversationID, accountUserID string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.leave")
	defer span.End()
	err := r.db.LeaveConversation(ctx, sqlc.LeaveConversationParams{
		ConversationID: conversationID, AccountUserID: db.NullStringPtr(&accountUserID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *participantRepoImpl) Hide(ctx context.Context, conversationID, accountUserID string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.hide")
	defer span.End()
	err := r.db.HideConversation(ctx, sqlc.HideConversationParams{
		ConversationID: conversationID, AccountUserID: db.NullStringPtr(&accountUserID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *participantRepoImpl) Unhide(ctx context.Context, conversationID, accountUserID string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.unhide")
	defer span.End()
	err := r.db.UnhideConversation(ctx, sqlc.UnhideConversationParams{
		ConversationID: conversationID, AccountUserID: db.NullStringPtr(&accountUserID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *participantRepoImpl) Reactivate(ctx context.Context, conversationID, accountUserID, role string) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.reactivate")
	defer span.End()
	err := r.db.ReactivateParticipant(ctx, sqlc.ReactivateParticipantParams{
		Role: role, ConversationID: conversationID, AccountUserID: db.NullStringPtr(&accountUserID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (r *participantRepoImpl) SetMute(ctx context.Context, conversationID, accountUserID string, isMuted bool, mutedUntil *time.Time) *apierror.APIError {
	ctx, span := conversationRepoTracer.Start(ctx, "repository.participant.set_mute")
	defer span.End()
	notifications := string(constants.ParticipantNotificationsUnmuted)
	if isMuted {
		notifications = string(constants.ParticipantNotificationsMuted)
	}
	err := r.db.SetParticipantNotifications(ctx, sqlc.SetParticipantNotificationsParams{
		Notifications: notifications, MutedUntil: db.NullTimePtr(mutedUntil), ConversationID: conversationID, AccountUserID: db.NullStringPtr(&accountUserID),
	})
	if apiErr := db.MapSQLError(err); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func conversationFromRow(row sqlc.Conversation) *domain.Conversation {
	return &domain.Conversation{
		ID:                   row.ID,
		AccountID:            row.AccountID,
		Type:                 row.Type,
		Audience:             row.Audience,
		Title:                db.StringFromNullString(row.Title),
		GroupID:              db.StringFromNullString(row.GroupID),
		TopicResourceType:    db.StringFromNullString(row.TopicResourceType),
		TopicResourceID:      db.StringFromNullString(row.TopicResourceID),
		NextSequence:         row.NextSequence,
		LastMessageID:        db.StringFromNullString(row.LastMessageID),
		LastMessageAt:        db.TimeFromNullTime(row.LastMessageAt),
		IsArchived:           row.IsArchived,
		LegalHold:            row.LegalHold,
		WorkflowStatus:       db.StringFromNullString(row.WorkflowStatus),
		AssigneeResourceType: db.StringFromNullString(row.AssigneeResourceType),
		AssigneeResourceID:   db.StringFromNullString(row.AssigneeResourceID),
		EmailInboxID:         db.StringFromNullString(row.EmailInboxID),
		Metadata:             json.RawMessage(row.Metadata),
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func participantFromRow(row sqlc.ConversationParticipant) *domain.ConversationParticipant {
	return &domain.ConversationParticipant{
		ID:                   row.ID,
		ConversationID:       row.ConversationID,
		AccountID:            row.AccountID,
		ParticipantType:      row.ParticipantType,
		AccountUserID:        db.StringFromNullString(row.AccountUserID),
		AgentConfigID:        db.StringFromNullString(row.AgentConfigID),
		Role:                 row.Role,
		Membership:           row.Membership,
		Notifications:        row.Notifications,
		LastReadSequence:     row.LastReadSequence,
		LastReadMessageID:    db.StringFromNullString(row.LastReadMessageID),
		LastReadAt:           db.TimeFromNullTime(row.LastReadAt),
		JoinedAt:             row.JoinedAt,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
		HiddenAt:             db.TimeFromNullTime(row.HiddenAt),
		AgentTriggerPolicy:   db.StringFromNullString(row.AgentTriggerPolicy),
		AgentTriggerKeywords: decodeKeywords(row.AgentTriggerKeywords),
		RelationAccountID:    db.StringFromNullString(row.RelationAccountID),
	}
}

// decodeKeywords parses the agent_trigger_keywords JSON array (NULL/invalid → nil).
func decodeKeywords(raw db.NullableRawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}
