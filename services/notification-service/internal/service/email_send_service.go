package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/ptrutil"
	"github.com/open-mrp/api/shared/tracing"
)

// SendInboxReply sends an agent's outbound email through the conversation's bound inbox and records it in the thread + ledger. Recipient and threading are derived from the latest inbound mail, so the agent cannot redirect the reply to an arbitrary address.
func (s *conversationSvcImpl) SendInboxReply(ctx context.Context, in domain.SendInboxReplyInput) (*domain.Message, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.send_inbox_reply")
	defer span.End()

	part, apiErr := s.resolveAgentReplyParticipant(ctx, in.ConversationID, in.AgentConfigID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if part == nil {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("The agent is not a participant in this conversation."))
	}
	return s.deliverInboxReply(ctx, in.ConversationID, part, in.AgentConfigID, in.AgentRunID, in.Body, in.Subject, in.Cc)
}

// deliverInboxReply is the shared outbound-email core for both agent (SendInboxReply) and human
// (ReplyToCustomer) customer replies: it sends via SES (recipient + threading derived from the latest inbound mail, never client-supplied), then records a visibility=external message + email_message ledger row in the same transaction. The customer sees the branded "Customer Service" party (applied at read time), never the individual staff author.
func (s *conversationSvcImpl) deliverInboxReply(ctx context.Context, conversationID string, senderPart *domain.ConversationParticipant, agentConfigID, agentRunID string, body, subject string, cc []string) (*domain.Message, *apierror.APIError) {
	if s.bridgeEmailSender == nil {
		return nil, apierror.NewInternalError(nil, "Email sending is not configured.")
	}
	if strings.TrimSpace(body) == "" {
		return nil, apierror.NewParameterMissingError("An email body is required.", "body")
	}

	// The latest inbound email anchors the reply: who to reply to + the threading headers.
	latest, apiErr := s.repoFactory.NewEmailMessageRepo().GetLatestInbound(ctx, conversationID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apierror.NewParameterInvalidError("This conversation has no inbound email to reply to.", "conversation_id")
		}
		return nil, apiErr
	}
	inbox, apiErr := s.repoFactory.NewEmailInboxRepo().GetByID(ctx, latest.EmailInboxID, latest.AccountID)
	if apiErr != nil {
		return nil, apiErr
	}
	accountID := inbox.AccountID

	messageID, apiErr := id.GenID(id.MessageIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	emID, apiErr := id.GenID(id.EmailMessageIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}

	// Outbound rfc Message-ID, on the inbox's domain so replies thread back to us.
	rfcMessageID := fmt.Sprintf("%s@%s", emID, addressDomain(inbox.Address))
	references := buildReferences(ptrutil.Deref(latest.References), latest.RfcMessageID)
	to := latest.FromAddr

	sesMessageID, apiErr := s.bridgeEmailSender.Send(ctx, domain.EmailData{
		To:         []string{to},
		Cc:         cc,
		Subject:    subject,
		Body:       body,
		From:       new(fromHeader(inbox)),
		InReplyTo:  &latest.RfcMessageID,
		References: &references,
		MessageID:  &rfcMessageID,
		PlainText:  true,
	})
	if apiErr != nil {
		return nil, apiErr
	}

	var result *domain.Message
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		convRepo := f.NewConversationRepo()
		seq, lockErr := convRepo.LockSequence(txCtx, conversationID, accountID)
		if lockErr != nil {
			return lockErr
		}
		now := time.Now()
		msgBody := body
		msg := &domain.Message{
			ID:                  messageID,
			ConversationID:      conversationID,
			AccountID:           accountID,
			Sequence:            seq,
			Kind:                string(constants.MessageKindEmail),
			Visibility:          string(constants.MessageVisibilityExternal),
			Channel:             new(string(constants.MessageChannelEmail)),
			SenderParticipantID: &senderPart.ID,
			AgentRunID:          strPtrIfNotEmpty(agentRunID),
			Body:                &msgBody,
			Preview:             strPtrIfNotEmpty(messagePreview(&msgBody, 0, false)),
			StreamingState:      new(messageStreamingStateComplete),
			CreatedAt:           now,
			SenderAgentConfigID: strPtrIfNotEmpty(agentConfigID),
		}
		if _, createErr := f.NewMessageRepo().Create(txCtx, msg); createErr != nil {
			return createErr
		}
		if advErr := convRepo.AdvanceAfterMessage(txCtx, conversationID, messageID, now); advErr != nil {
			return advErr
		}
		var sesID *string
		if sesMessageID != nil {
			sesID = sesMessageID
		}
		if _, emErr := f.NewEmailMessageRepo().TryInsert(txCtx, &domain.CreateEmailMessageInput{
			ID:             emID,
			AccountID:      accountID,
			ConversationID: conversationID,
			MessageID:      messageID,
			EmailInboxID:   inbox.ID,
			Direction:      domain.EmailDirectionOutbound,
			RfcMessageID:   rfcMessageID,
			InReplyTo:      &latest.RfcMessageID,
			References:     &references,
			FromAddr:       inbox.Address,
			ToAddrs:        to,
			CcAddrs:        strPtrIfNotEmpty(strings.Join(cc, ", ")),
			Subject:        &subject,
			SesMessageID:   sesID,
		}); emErr != nil {
			return emErr
		}
		if rtErr := s.fanoutMessageRealtime(txCtx, f, conversationID, accountID, messageID, seq, string(constants.MessageVisibilityExternal)); rtErr != nil {
			return rtErr
		}
		result = msg
		return nil
	})
	if apiErr != nil {
		return nil, apiErr
	}
	s.kickOutbox()
	return result, nil
}

// PostReplyDraft proposes an agent's outbound reply to a case's external party (customer, supplier, or
// other contact) as a real status=draft message, held internal-visible until a human approves it via the
// reply-drafts bar (which then promotes it over the resolved channel — email if the case is bridged, else
// a portal message). This is the genuine draft the approval UI acts on (created via CreateDraft,
// status=draft), backing the draft_reply tool.
func (s *conversationSvcImpl) PostReplyDraft(ctx context.Context, in domain.PostReplyDraftInput) (*domain.Message, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.post_reply_draft")
	defer span.End()

	body := strings.TrimSpace(in.Body)
	if body == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("A draft body is required.", "body"))
	}
	// The draft is authored by the agent participant; a removed/inactive agent can't propose a reply.
	part, apiErr := s.resolveAgentReplyParticipant(ctx, in.ConversationID, in.AgentConfigID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if part == nil {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("The agent is not a participant in this conversation."))
	}
	accountID := part.AccountID
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, in.ConversationID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Drafts are an external-case affordance; on an internal conversation there's no external party to send to.
	if conv.Audience != string(constants.ConversationAudienceCustomer) {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("Reply drafts are only supported on external cases.", "conversation_id"))
	}
	messageID, apiErr := id.GenID(id.MessageIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// A bridged case sends by email; an unbridged (portal) case sends an in-conversation message. The
	// channel is what approval promotes the draft over (see promoteDraftViaEmail/Portal).
	channel := string(constants.MessageChannelMessage)
	if conv.EmailInboxID != nil && *conv.EmailInboxID != "" {
		channel = string(constants.MessageChannelEmail)
	}
	draft := &domain.Message{
		ID:                    messageID,
		AccountID:             accountID,
		ConversationID:        in.ConversationID,
		Channel:               &channel,
		Subject:               strPtrIfNotEmpty(in.Subject),
		Body:                  &body,
		Preview:               strPtrIfNotEmpty(messagePreview(&body, 0, false)),
		SenderParticipantID:   &part.ID,
		AgentRunID:            strPtrIfNotEmpty(in.AgentRunID),
		SourceThreadMessageID: strPtrIfNotEmpty(in.SourceThreadMessageID),
	}
	var result *domain.Message
	if apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		msgRepo := f.NewMessageRepo()
		if cErr := msgRepo.CreateDraft(txCtx, draft); cErr != nil {
			return cErr
		}
		loaded, lErr := msgRepo.GetByID(txCtx, messageID, accountID)
		if lErr != nil {
			return lErr
		}
		result = loaded
		return nil
	}); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// The proposed reply now awaits human approval; move the case into the needs-approval lane and nudge
	// its viewers so the reply-drafts bar (and inbox lane) refresh live.
	s.autoSetCaseWorkflow(ctx, in.ConversationID, accountID, constants.ConversationWorkflowStatusNeedsApproval)
	s.fanoutConversationEvent(ctx, in.ConversationID, accountID, "conversation.updated", "")
	s.kickOutbox()
	return result, nil
}

// fromHeader renders an inbox's outbound From value, preferring "Name <addr>" when a display name is set.
func fromHeader(inbox *domain.EmailInbox) string {
	if inbox.FromName != nil && *inbox.FromName != "" {
		return fmt.Sprintf("%s <%s>", *inbox.FromName, inbox.Address)
	}
	return inbox.Address
}

// addressDomain returns the domain part of an email address (after the last @), or the mail
// domain if absent. It supplies the domain half of an RFC Message-ID, so the fallback tracks
// where mail is actually sent from — augno.com, which is what SES has verified — rather than
// the web domain.
func addressDomain(addr string) string {
	if i := strings.LastIndex(addr, "@"); i >= 0 && i < len(addr)-1 {
		return addr[i+1:]
	}
	return "augno.com"
}

// buildReferences appends the just-replied-to message-id to the prior References chain, de-duplicated.
func buildReferences(priorRefs string, latestID string) string {
	seen := map[string]bool{}
	var out []string
	for _, r := range strings.Fields(priorRefs) {
		r = strings.Trim(r, "<>")
		if r != "" && !seen[r] {
			seen[r] = true
			out = append(out, r)
		}
	}
	if latestID != "" && !seen[latestID] {
		out = append(out, latestID)
	}
	return strings.Join(out, " ")
}
