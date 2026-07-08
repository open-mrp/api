package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/ptrutil"
	"github.com/augno/api/shared/tracing"
)

// supportAliasDisplayName is the branded party a customer sees behind every staff reply on an external
// case. The customer never sees the individual operator: staff/agent authors are collapsed to this
// single "Customer Service" alias at read time (see resolveSenders), so nothing needs to be persisted.
const supportAliasDisplayName = "Customer Service"

// UpdateWorkflowStatus sets an external case's triage lane.
func (s *conversationSvcImpl) UpdateWorkflowStatus(ctx context.Context, conversationID, status string) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.update_workflow_status")
	defer span.End()

	if !constants.ConversationWorkflowStatus(status).IsValid() {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("Unknown workflow status.", "status"))
	}
	accountID, apiErr := s.requireMessagingAdmin(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	conv, apiErr := s.requireCustomerCase(ctx, conversationID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Pre-image captured before the write; the copy applies the new status so the audit diff surfaces workflow_status.
	updated := *conv
	updated.WorkflowStatus = &status
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if apiErr := f.NewConversationRepo().SetWorkflowStatus(txCtx, conv.ID, accountID, status); apiErr != nil {
			return apiErr
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeConversation,
			ResourceID:   conversationID,
			Changes:      audit.ComputeChanges(conv, &updated),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.loadConversationForAdmin(ctx, conversationID, accountID)
}

// AssignConversation sets or clears the owning user/team for an external case.
func (s *conversationSvcImpl) AssignConversation(ctx context.Context, conversationID string, assigneeResourceType, assigneeResourceID *string) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.assign")
	defer span.End()

	accountID, apiErr := s.requireMessagingAdmin(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	conv, apiErr := s.requireCustomerCase(ctx, conversationID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Pre-image captured before the write; the copy applies the new (or cleared) assignee so the audit diff
	// surfaces assignee_resource_type/id for both assignment and unassignment (nils).
	updated := *conv
	updated.AssigneeResourceType = assigneeResourceType
	updated.AssigneeResourceID = assigneeResourceID
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if apiErr := f.NewConversationRepo().Assign(txCtx, conv.ID, accountID, assigneeResourceType, assigneeResourceID); apiErr != nil {
			return apiErr
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeConversation,
			ResourceID:   conversationID,
			Changes:      audit.ComputeChanges(conv, &updated),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.loadConversationForAdmin(ctx, conversationID, accountID)
}

// ListInbox lists external customer-facing cases for the support inbox.
func (s *conversationSvcImpl) ListInbox(ctx context.Context, input domain.SupportInboxInput) (*domain.ConversationPage, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.list_inbox")
	defer span.End()

	accountID, apiErr := s.requireMessagingAdmin(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if input.WorkflowStatus != nil && !constants.ConversationWorkflowStatus(*input.WorkflowStatus).IsValid() {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("Unknown workflow status.", "status"))
	}
	limit := clampLimit(input.Limit, defaultNotificationPageSize, maxNotificationPageSize)
	cursorAt, cursorID, apiErr := decodeCursor(input.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rows, apiErr := s.repoFactory.NewConversationRepo().ListInbox(ctx, domain.SupportInboxFilter{
		AccountID:           accountID,
		WorkflowStatus:      input.WorkflowStatus,
		AssigneeResourceID:  input.AssigneeResourceID,
		Unassigned:          input.Unassigned,
		IncludeArchived:     input.IncludeArchived,
		Limit:               limit + 1,
		CursorLastMessageAt: cursorAt,
		CursorID:            cursorID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	page := &domain.ConversationPage{}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		page.HasNextPage = true
		last := rows[len(rows)-1]
		if last.LastMessageAt != nil {
			next := encodeCursor(*last.LastMessageAt, last.ID)
			page.NextCursor = &next
		}
	}
	// Hydrate participants + last-message preview for the page (staff view: the actual last message).
	partRepo := s.repoFactory.NewParticipantRepo()
	lastMessageIDs := make([]string, 0, len(rows))
	for _, conv := range rows {
		participants, apiErr := partRepo.List(ctx, conv.ID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		conv.Participants = participants
		if conv.LastMessageID != nil && *conv.LastMessageID != "" {
			lastMessageIDs = append(lastMessageIDs, *conv.LastMessageID)
		}
	}
	if len(lastMessageIDs) > 0 {
		msgs, apiErr := s.repoFactory.NewMessageRepo().GetByIDs(ctx, lastMessageIDs)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		byID := make(map[string]*domain.Message, len(msgs))
		for _, m := range msgs {
			byID[m.ID] = m
		}
		for _, conv := range rows {
			if conv.LastMessageID != nil {
				if m, ok := byID[*conv.LastMessageID]; ok {
					conv.LastMessage = m
				}
			}
		}
	}
	s.hydrateConversationListPresentation(ctx, rows)
	page.Conversations = rows
	return page, nil
}

// AddConversationLink links a business record to a conversation the caller can administer.
func (s *conversationSvcImpl) AddConversationLink(ctx context.Context, conversationID, resourceType, resourceID string) (*domain.ConversationLink, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.add_link")
	defer span.End()

	if resourceType == "" || resourceID == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("A resource_type and resource_id are required.", "resource_type"))
	}
	accountID, part, apiErr := s.requireCaseAdmin(ctx, conversationID, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	linkID, apiErr := id.GenID(id.ConversationLinkIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	link := &domain.ConversationLink{
		ID:                     linkID,
		AccountID:              accountID,
		ConversationID:         conversationID,
		ResourceType:           resourceType,
		ResourceID:             resourceID,
		CreatedByParticipantID: part,
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if apiErr := f.NewConversationLinkRepo().Create(txCtx, link); apiErr != nil {
			return apiErr
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeConversationLink,
			ResourceID:   link.ID,
			Changes:      audit.ComputeChanges(nil, link),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return link, nil
}

// RemoveConversationLink removes a business-record link from a conversation by link id.
func (s *conversationSvcImpl) RemoveConversationLink(ctx context.Context, conversationID, linkID string) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.remove_link")
	defer span.End()

	accountID, _, apiErr := s.requireCaseAdmin(ctx, conversationID, types.ActionUpdate)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		removed, apiErr := f.NewConversationLinkRepo().Delete(txCtx, linkID, conversationID, accountID)
		if apiErr != nil {
			return apiErr
		}
		if !removed {
			return apierror.NewResourceNotFoundError("Link not found.")
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeConversationLink,
			ResourceID:   linkID,
		})
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// ListConversationLinks lists a conversation's secondary business-record links.
func (s *conversationSvcImpl) ListConversationLinks(ctx context.Context, conversationID string) ([]*domain.ConversationLink, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.list_links")
	defer span.End()

	accountID, _, apiErr := s.requireCaseAdmin(ctx, conversationID, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewConversationLinkRepo().List(ctx, conversationID, accountID)
}

// ListConversationsByResource lists conversations linked to a business record (topic anchor or link).
func (s *conversationSvcImpl) ListConversationsByResource(ctx context.Context, resourceType, resourceID string, limit int32) ([]*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.list_by_resource")
	defer span.End()

	if resourceType == "" || resourceID == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("A resource_type and resource_id are required.", "resource_type"))
	}
	accountID, apiErr := s.requireMessagingAdmin(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewConversationRepo().ListByResource(ctx, accountID, resourceType, resourceID, clampLimit(limit, defaultNotificationPageSize, maxNotificationPageSize))
}

// CreateReplyDraft proposes a customer-reply draft on an external case (draft-first; not sent). The
// draft is a message row at status=draft, held internal-visible until approved.
func (s *conversationSvcImpl) CreateReplyDraft(ctx context.Context, input domain.CreateReplyDraftInput) (*domain.Message, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.create_reply_draft")
	defer span.End()

	if !constants.MessageChannel(input.Channel).IsValid() {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("Unknown message channel.", "channel"))
	}
	if input.Body == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("A draft body is required.", "body"))
	}
	accountID, part, apiErr := s.requireCaseAdmin(ctx, input.ConversationID, types.ActionCreate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if _, apiErr := s.requireCustomerCase(ctx, input.ConversationID, accountID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Recovery-point idempotency: a draft mints a fresh id with no dedup key, so a retry would create a duplicate draft.
	identity, _ := appctx.GetIdentityFromContext(ctx)
	idemKey, apiErr := upsertIdempotencyKey(ctx, s.repoFactory, identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	switch domain.RecoveryPoint(idemKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.Message](ctx, idemKey.ResponseCode, idemKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error
	case domain.RecoveryPointStarted:
		draftID, apiErr := id.GenID(id.MessageIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		channel := input.Channel
		body := input.Body
		draft := &domain.Message{
			ID:                    draftID,
			AccountID:             accountID,
			ConversationID:        input.ConversationID,
			Channel:               &channel,
			Subject:               input.Subject,
			Body:                  &body,
			Preview:               strPtrIfNotEmpty(messagePreview(&body, 0, false)),
			SenderParticipantID:   part,
			AgentRunID:            strPtrIfNotEmpty(input.AgentRunID),
			SourceThreadMessageID: input.SourceThreadMessageID,
		}
		var result *domain.Message
		apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
			msgRepo := f.NewMessageRepo()
			if cErr := msgRepo.CreateDraft(txCtx, draft); cErr != nil {
				return cErr
			}
			loaded, lErr := msgRepo.GetByID(txCtx, draftID, accountID)
			if lErr != nil {
				return lErr
			}
			result = loaded
			return cacheSuccessResponse(txCtx, f, idemKey.TypeID, result)
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, cacheErrorResponse(ctx, s.repoFactory, idemKey.TypeID, apiErr))
		}
		// A proposed reply now awaits human approval before it can reach the customer.
		s.autoSetCaseWorkflow(ctx, input.ConversationID, accountID, constants.ConversationWorkflowStatusNeedsApproval)
		return result, nil
	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idemKey.RecoveryPoint))
	}
}

// ListReplyDrafts lists a case's reply drafts (optionally filtered by status).
func (s *conversationSvcImpl) ListReplyDrafts(ctx context.Context, conversationID string, status *string) ([]*domain.Message, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.list_reply_drafts")
	defer span.End()

	accountID, _, apiErr := s.requireCaseAdmin(ctx, conversationID, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewMessageRepo().ListDrafts(ctx, conversationID, accountID, status)
}

// UpdateReplyDraft edits a still-open draft's body/subject.
func (s *conversationSvcImpl) UpdateReplyDraft(ctx context.Context, draftID, body string, subject *string) (*domain.Message, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.update_reply_draft")
	defer span.End()

	if body == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("A draft body is required.", "body"))
	}
	accountID, apiErr := s.requireMessagingAdmin(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	repo := s.repoFactory.NewMessageRepo()
	draft, apiErr := repo.GetByID(ctx, draftID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if draft.Status != string(constants.MessageStatusDraft) {
		return nil, tracing.Trace(span, apierror.NewResourceConflictError("This draft can no longer be edited."))
	}
	preview := strPtrIfNotEmpty(messagePreview(&body, 0, false))
	if apiErr := repo.UpdateDraftContent(ctx, draftID, accountID, body, subject, preview); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return repo.GetByID(ctx, draftID, accountID)
}

// RejectReplyDraft discards an open draft without sending.
func (s *conversationSvcImpl) RejectReplyDraft(ctx context.Context, draftID string) (*domain.Message, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.reject_reply_draft")
	defer span.End()

	accountID, apiErr := s.requireMessagingAdmin(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	repo := s.repoFactory.NewMessageRepo()
	applied, apiErr := repo.SetDraftStatus(ctx, draftID, accountID, string(constants.MessageStatusRejected))
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !applied {
		return nil, tracing.Trace(span, apierror.NewResourceConflictError("This draft is no longer open."))
	}
	draft, apiErr := repo.GetByID(ctx, draftID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// The proposed reply was discarded — the team still owes the customer a response.
	s.autoSetCaseWorkflow(ctx, draft.ConversationID, accountID, constants.ConversationWorkflowStatusWaitingInternal)
	return draft, nil
}

// ApproveAndSendReplyDraft promotes an open draft to a sent customer-visible message in place (portal
// or email), attributed to the case's alias persona. The promote is a compare-and-set on status='draft'
// so a concurrent double-approve sends exactly once.
func (s *conversationSvcImpl) ApproveAndSendReplyDraft(ctx context.Context, draftID, clientMessageID string) (*domain.Message, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.approve_send_reply_draft")
	defer span.End()

	identity, approverAcus, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if identity.IsRelationActor() {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("This operation is not available to customer accounts."))
	}
	repo := s.repoFactory.NewMessageRepo()
	draft, apiErr := repo.GetByID(ctx, draftID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if draft.Status != string(constants.MessageStatusDraft) {
		return nil, tracing.Trace(span, apierror.NewResourceConflictError("This draft is no longer open."))
	}
	conv, apiErr := s.requireCustomerCase(ctx, draft.ConversationID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	channel := ""
	if draft.Channel != nil {
		channel = *draft.Channel
	}
	if channel == string(constants.MessageChannelEmail) && conv.EmailInboxID != nil && *conv.EmailInboxID != "" {
		apiErr = s.promoteDraftViaEmail(ctx, draft, conv, approverAcus)
	} else {
		apiErr = s.promoteDraftViaPortal(ctx, draft, approverAcus)
	}
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// The team just replied to the customer — the ball is now in the customer's court.
	s.autoSetCaseWorkflow(ctx, conv.ID, accountID, constants.ConversationWorkflowStatusWaitingExternal)
	return repo.GetByID(ctx, draftID, accountID)
}

// promoteDraftViaPortal promotes a draft to a sent customer-visible portal message in place: assigns
// the conversation sequence, flips it to customer visibility under the persona, and fans it out.
func (s *conversationSvcImpl) promoteDraftViaPortal(ctx context.Context, draft *domain.Message, approverAcus string) *apierror.APIError {
	apiErr := s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		convRepo := f.NewConversationRepo()
		seq, lockErr := convRepo.LockSequence(txCtx, draft.ConversationID, draft.AccountID)
		if lockErr != nil {
			return lockErr
		}
		applied, promErr := f.NewMessageRepo().PromoteDraft(txCtx, draft.ID, draft.AccountID, string(constants.MessageKindChat), seq, &approverAcus, draft.Preview)
		if promErr != nil {
			return promErr
		}
		if !applied {
			// A concurrent approve already sent it; idempotent no-op.
			return nil
		}
		now := time.Now()
		draft.Sequence = seq
		draft.Status = string(constants.MessageStatusSent)
		draft.Visibility = string(constants.MessageVisibilityExternal)
		if advErr := convRepo.AdvanceAfterMessage(txCtx, draft.ConversationID, draft.ID, now); advErr != nil {
			return advErr
		}
		if rtErr := s.fanoutMessageRealtime(txCtx, f, draft.ConversationID, draft.AccountID, draft.ID, seq, string(constants.MessageVisibilityExternal)); rtErr != nil {
			return rtErr
		}
		return s.fanoutMessageNotifications(txCtx, f, draft, draft.AccountID, "")
	})
	if apiErr != nil {
		return apiErr
	}
	s.kickOutbox()
	return nil
}

// promoteDraftViaEmail sends the draft as outbound mail through the case's bridged inbox (SES, threaded
// under the latest inbound mail) and promotes the same draft row to a sent customer-visible email
// message in place, recording it in the email_message ledger — one resource, not a separate row.
func (s *conversationSvcImpl) promoteDraftViaEmail(ctx context.Context, draft *domain.Message, conv *domain.Conversation, approverAcus string) *apierror.APIError {
	if s.bridgeEmailSender == nil {
		return apierror.NewInternalError(nil, "Email sending is not configured.")
	}
	body := ""
	if draft.Body != nil {
		body = *draft.Body
	}
	subject := caseEmailSubject(draft.Subject, conv.Title)

	// The latest inbound email anchors the reply: who to reply to + the threading headers.
	latest, apiErr := s.repoFactory.NewEmailMessageRepo().GetLatestInbound(ctx, draft.ConversationID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return apierror.NewParameterInvalidError("This conversation has no inbound email to reply to.", "conversation_id")
		}
		return apiErr
	}
	inbox, apiErr := s.repoFactory.NewEmailInboxRepo().GetByID(ctx, latest.EmailInboxID, latest.AccountID)
	if apiErr != nil {
		return apiErr
	}
	accountID := inbox.AccountID
	emID, apiErr := id.GenID(id.EmailMessageIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}
	rfcMessageID := fmt.Sprintf("%s@%s", emID, addressDomain(inbox.Address))
	references := buildReferences(ptrutil.Deref(latest.References), latest.RfcMessageID)
	to := latest.FromAddr

	sesMessageID, apiErr := s.bridgeEmailSender.Send(ctx, domain.EmailData{
		To:         []string{to},
		Subject:    subject,
		Body:       body,
		From:       new(fromHeader(inbox)),
		InReplyTo:  &latest.RfcMessageID,
		References: &references,
		MessageID:  &rfcMessageID,
		PlainText:  true,
	})
	if apiErr != nil {
		return apiErr
	}

	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		convRepo := f.NewConversationRepo()
		seq, lockErr := convRepo.LockSequence(txCtx, draft.ConversationID, accountID)
		if lockErr != nil {
			return lockErr
		}
		// Promote the draft row in place: status='sent', visibility=external, kind=email, sequence.
		applied, promErr := f.NewMessageRepo().PromoteDraft(txCtx, draft.ID, accountID, string(constants.MessageKindEmail), seq, &approverAcus, draft.Preview)
		if promErr != nil {
			return promErr
		}
		if !applied {
			// A concurrent approve already sent it; idempotent no-op (the SES send above is the only
			// possible duplicate, guarded by the draft-state compare-and-set on the winning approve).
			return nil
		}
		now := time.Now()
		draft.Sequence = seq
		draft.Status = string(constants.MessageStatusSent)
		draft.Visibility = string(constants.MessageVisibilityExternal)
		if advErr := convRepo.AdvanceAfterMessage(txCtx, draft.ConversationID, draft.ID, now); advErr != nil {
			return advErr
		}
		if _, emErr := f.NewEmailMessageRepo().TryInsert(txCtx, &domain.CreateEmailMessageInput{
			ID:             emID,
			AccountID:      accountID,
			ConversationID: draft.ConversationID,
			MessageID:      draft.ID,
			EmailInboxID:   inbox.ID,
			Direction:      domain.EmailDirectionOutbound,
			RfcMessageID:   rfcMessageID,
			InReplyTo:      &latest.RfcMessageID,
			References:     &references,
			FromAddr:       inbox.Address,
			ToAddrs:        to,
			Subject:        &subject,
			SesMessageID:   sesMessageID,
		}); emErr != nil {
			return emErr
		}
		return s.fanoutMessageRealtime(txCtx, f, draft.ConversationID, accountID, draft.ID, seq, string(constants.MessageVisibilityExternal))
	})
	if apiErr != nil {
		return apiErr
	}
	s.kickOutbox()
	return nil
}

// requireCustomerCase loads a conversation for an account-level admin and asserts it is an external
// (audience=customer) case. Returns a not-found rather than leaking an internal conversation's existence.
func (s *conversationSvcImpl) requireCustomerCase(ctx context.Context, conversationID, accountID string) (*domain.Conversation, *apierror.APIError) {
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, accountID)
	if apiErr != nil {
		return nil, apiErr
	}
	if conv.Audience != string(constants.ConversationAudienceCustomer) {
		return nil, apierror.NewParameterInvalidError("This conversation is not a customer-facing case.", "conversation_id")
	}
	return conv, nil
}

// requireCaseAdmin authorizes a conversation-scoped admin action: an internal actor in the account holding the messaging permission, with the conversation present. Returns (accountID, caller participant id if a member else nil, err). Links/drafts may target internal object-linked conversations too, so it does not require an audience.
func (s *conversationSvcImpl) requireCaseAdmin(ctx context.Context, conversationID string, action types.Action) (string, *string, *apierror.APIError) {
	accountID, apiErr := s.requireMessagingAdmin(ctx, action)
	if apiErr != nil {
		return "", nil, apiErr
	}
	if _, apiErr := s.loadConversationForAdmin(ctx, conversationID, accountID); apiErr != nil {
		return "", nil, apiErr
	}
	// Best-effort caller participant id (for attribution); nil when the admin is not a member.
	var partID *string
	if _, callerAcus, _, cErr := s.caller(ctx); cErr == nil {
		if p, pErr := s.repoFactory.NewParticipantRepo().Get(ctx, conversationID, callerAcus); pErr == nil {
			partID = &p.ID
		}
	}
	return accountID, partID, nil
}

// autoSetCaseWorkflow best-effort moves an external case into a triage lane in response to activity (a staff reply → waiting on customer, a draft → needs approval, …). It never fails the triggering action:
// a status-write hiccup must not block the message, reply, or draft that caused it.
func (s *conversationSvcImpl) autoSetCaseWorkflow(ctx context.Context, conversationID, accountID string, status constants.ConversationWorkflowStatus) {
	if apiErr := s.repoFactory.NewConversationRepo().SetWorkflowStatus(ctx, conversationID, accountID, string(status)); apiErr != nil {
		slog.WarnContext(ctx, "auto case workflow advance failed", "conversation_id", conversationID, "status", string(status), "error", apiErr)
	}
}

// advanceCaseOnCustomerInbound moves an external case to "waiting on team" when the customer sends a message, so the inbox reflects that the team owes a reply (and a resolved case reopens). No-op for internal conversations. Best-effort: never blocks the inbound message.
func (s *conversationSvcImpl) advanceCaseOnCustomerInbound(ctx context.Context, conversationID, accountID string) {
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, accountID)
	if apiErr != nil {
		slog.WarnContext(ctx, "auto case workflow: load failed", "conversation_id", conversationID, "error", apiErr)
		return
	}
	if conv.Audience != string(constants.ConversationAudienceCustomer) {
		return
	}
	s.autoSetCaseWorkflow(ctx, conversationID, accountID, constants.ConversationWorkflowStatusWaitingInternal)
}

// caseEmailSubject resolves the outbound subject for an email reply: the explicit subject, else a
// "Re: <case title>" derived from the conversation, else empty (threading still works via headers).
func caseEmailSubject(explicit, title *string) string {
	if explicit != nil && *explicit != "" {
		return *explicit
	}
	if title != nil && *title != "" {
		return "Re: " + *title
	}
	return ""
}
