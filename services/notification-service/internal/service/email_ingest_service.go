package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/tracing"
)

// IngestInboundEmail threads a parsed inbound email into the conversation bound to its inbox, records it in the email_message ledger for at-least-once dedup, and dispatches it to the inbox's agent.
// It is a system operation (no caller identity): the account is taken from the resolved inbox.
func (s *conversationSvcImpl) IngestInboundEmail(ctx context.Context, in domain.IngestInboundEmailInput) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.ingest_inbound_email")
	defer span.End()

	if in.RfcMessageID == "" {
		return tracing.Trace(span, apierror.NewParameterMissingError("An rfc message id is required.", "rfc_message_id"))
	}

	// Resolve the inbox the mail was delivered to, trying every candidate recipient. Unknown/disabled
	// inboxes are dropped (acked) — a catch-all SES rule can deliver mail for addresses we don't host a
	// thread for.
	inbox, apiErr := s.resolveInbox(ctx, in.Recipients)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if inbox == nil {
		slog.WarnContext(ctx, "inbound email for unknown inbox dropped", "recipients", in.Recipients)
		return nil
	}
	if inbox.Status != domain.EmailInboxStatusActive {
		slog.WarnContext(ctx, "inbound email for disabled inbox dropped", "inbox", inbox.ID)
		return nil
	}
	accountID := inbox.AccountID

	// Redelivery dedup: if this rfc Message-ID is already recorded, ack and skip.
	if _, apiErr := s.repoFactory.NewEmailMessageRepo().GetByRfcID(ctx, in.RfcMessageID); apiErr == nil {
		return nil
	} else if apiErr.Code != apierror.ErrorCodeResourceNotFound {
		return tracing.Trace(span, apiErr)
	}

	// Thread resolution: In-Reply-To + References point at prior emails; match one to its conversation.
	conversationID := ""
	if candidates := threadCandidates(in.InReplyTo, in.References); len(candidates) > 0 {
		match, apiErr := s.repoFactory.NewEmailMessageRepo().FindThreadConversation(ctx, candidates)
		if apiErr == nil {
			conversationID = match.ConversationID
		} else if apiErr.Code != apierror.ErrorCodeResourceNotFound {
			return tracing.Trace(span, apiErr)
		}
	}

	messageID, apiErr := id.GenID(id.MessageIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	emID, apiErr := id.GenID(id.EmailMessageIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	ingestedNew := false
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		convRepo := f.NewConversationRepo()

		// First message of a thread: create the inbox-bound conversation + agent participant.
		if conversationID == "" {
			newID, createErr := s.createEmailThreadConversation(txCtx, f, inbox, in)
			if createErr != nil {
				return createErr
			}
			conversationID = newID
		}

		seq, lockErr := convRepo.LockSequence(txCtx, conversationID, accountID)
		if lockErr != nil {
			return lockErr
		}

		now := time.Now()
		clientMsgID := in.RfcMessageID // message-level idempotency mirrors the rfc dedup
		body := in.TextBody
		msg := &domain.Message{
			ID:              messageID,
			ConversationID:  conversationID,
			AccountID:       accountID,
			Sequence:        seq,
			Kind:            string(constants.MessageKindEmail),
			Visibility:      string(constants.MessageVisibilityExternal),
			Channel:         constants.MessageChannelPtr(constants.MessageChannelEmail),
			ClientMessageID: &clientMsgID,
			Body:            &body,
			Preview:         strPtrIfNotEmpty(messagePreview(&body, 0, false)),
			CreatedAt:       now,
		}
		inserted, createErr := f.NewMessageRepo().Create(txCtx, msg)
		if createErr != nil {
			return createErr
		}
		if !inserted {
			// A message with this rfc id already exists in the thread — nothing more to do.
			return nil
		}
		if advErr := convRepo.AdvanceAfterMessage(txCtx, conversationID, messageID, now); advErr != nil {
			return advErr
		}

		// Ledger row (unique rfc_message_id) — the authoritative dedup guard. A concurrent delivery that raced past the pre-tx check loses here; we roll back (discarding a duplicate fresh thread).
		emInserted, emErr := f.NewEmailMessageRepo().TryInsert(txCtx, &domain.CreateEmailMessageInput{
			ID:             emID,
			AccountID:      accountID,
			ConversationID: conversationID,
			MessageID:      messageID,
			EmailInboxID:   inbox.ID,
			Direction:      domain.EmailDirectionInbound,
			RfcMessageID:   in.RfcMessageID,
			InReplyTo:      strPtrIfNotEmpty(in.InReplyTo),
			References:     strPtrIfNotEmpty(strings.Join(in.References, " ")),
			FromAddr:       in.From,
			ToAddrs:        inbox.Address,
			Subject:        strPtrIfNotEmpty(in.Subject),
			RawS3Key:       strPtrIfNotEmpty(in.RawS3Key),
		})
		if emErr != nil {
			return emErr
		}
		if !emInserted {
			return apierror.NewResourceConflictError("This email was already ingested.")
		}

		if rtErr := s.fanoutMessageRealtime(txCtx, f, conversationID, accountID, messageID, seq, string(constants.MessageVisibilityExternal)); rtErr != nil {
			return rtErr
		}
		// Fire the inbox's agent. The sender is external (not an agent), so dispatchAgents evaluates the agent participant's trigger policy exactly as it would for a human chat message.
		if dErr := s.dispatchAgents(txCtx, f, conversationID, accountID, string(constants.ParticipantTypeSystem), "", msg); dErr != nil {
			return dErr
		}
		ingestedNew = true
		return nil
	})
	if apiErr != nil {
		// A losing concurrent delivery rolled back; treat as success so the message is acked.
		if apiErr.Code == apierror.ErrorCodeResourceConflict {
			return nil
		}
		return tracing.Trace(span, apiErr)
	}

	// A newly-threaded inbound email puts the ball in the team's court (a freshly-created thread stays in its seeded "New" lane; an existing case advances to "waiting on team").
	if ingestedNew {
		s.advanceCaseOnCustomerInbound(ctx, conversationID, accountID)
	}

	s.kickOutbox()
	return nil
}

// resolveInbox finds the inbox an inbound mail belongs to by trying each candidate recipient in order.
// A candidate on the Augno receiving subdomain is a per-inbox forwarding address whose local part is the
// inbox id (resolved by id, no account scope); any other candidate is matched directly as an inbox
// address. The first candidate that resolves to a known inbox wins. A nil inbox with nil error means no
// candidate matched — the caller drops (acks) the mail. This tolerates forwarding, where the original
// inbox address survives only in To/Cc and the forwarding address only in the delivery headers.
func (s *conversationSvcImpl) resolveInbox(ctx context.Context, recipients []string) (*domain.EmailInbox, *apierror.APIError) {
	repo := s.repoFactory.NewEmailInboxRepo()
	forwardSuffix := ""
	if s.inboundEmailDomain != "" {
		forwardSuffix = "@" + strings.ToLower(strings.TrimSpace(s.inboundEmailDomain))
	}
	for _, r := range recipients {
		r = strings.ToLower(strings.TrimSpace(r))
		if r == "" {
			continue
		}
		var (
			inbox  *domain.EmailInbox
			apiErr *apierror.APIError
		)
		if forwardSuffix != "" && strings.HasSuffix(r, forwardSuffix) {
			// Per-inbox forwarding address: <inbox_id>@<inbound domain>. The local part is the inbox id.
			inbox, apiErr = repo.GetByIDSystem(ctx, strings.TrimSuffix(r, forwardSuffix))
		} else {
			inbox, apiErr = repo.GetByAddress(ctx, r)
		}
		if apiErr == nil {
			return inbox, nil
		}
		if apiErr.Code != apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
	}
	return nil, nil
}

// createEmailThreadConversation creates the conversation backing a new email thread: a group bound to the inbox, with the inbox's configured agent added as a participant (so it triages inbound mail).
func (s *conversationSvcImpl) createEmailThreadConversation(txCtx context.Context, f domain.RepoFactory, inbox *domain.EmailInbox, in domain.IngestInboundEmailInput) (string, *apierror.APIError) {
	conversationID, genErr := id.GenID(id.ConversationIDPrefix, nil)
	if genErr != nil {
		return "", genErr
	}
	title := strings.TrimSpace(in.Subject)
	if title == "" {
		title = "(no subject)"
	}
	if len(title) > 255 {
		title = title[:255]
	}
	// An email-bridged thread is an external customer-facing case: the customer's mail and the team's replies live in the customer-visible timeline, internal discussion as internal notes.
	convType := string(constants.ConversationTypeGroup)
	if createErr := f.NewConversationRepo().Create(txCtx, conversationID, &domain.CreateConversationInput{
		Type:     convType,
		Audience: string(constants.ConversationAudienceCustomer),
		Title:    &title,
	}, inbox.AccountID); createErr != nil {
		return "", createErr
	}
	if bindErr := f.NewConversationRepo().BindInbox(txCtx, conversationID, inbox.AccountID, inbox.ID, in.From); bindErr != nil {
		return "", bindErr
	}
	// Seed the triage lane so the case behaves like a portal support case (which is created 'new'):
	// it surfaces in the inbox's "New" lane until a human triages it, and inbound/reply activity can advance it from there. The generic conversation Create leaves workflow_status null.
	if wfErr := f.NewConversationRepo().SetWorkflowStatus(txCtx, conversationID, inbox.AccountID, string(constants.ConversationWorkflowStatusNew)); wfErr != nil {
		return "", wfErr
	}

	partRepo := f.NewParticipantRepo()
	seenAgents := map[string]bool{}
	seatAgent := func(agentConfigID, policy string, keywords []string) *apierror.APIError {
		if agentConfigID == "" || seenAgents[agentConfigID] {
			return nil
		}
		seenAgents[agentConfigID] = true
		pid, pgenErr := id.GenID(id.ConversationParticipantIDPrefix, nil)
		if pgenErr != nil {
			return pgenErr
		}
		return partRepo.CreateAgent(txCtx, pid, inbox.AccountID, &domain.AddAgentParticipantInput{
			ConversationID:  conversationID,
			AgentConfigID:   agentConfigID,
			TriggerPolicy:   policy,
			TriggerKeywords: keywords,
		})
	}

	// Seat the inbox's triage agent first. Email has no @mention, so the inbox's policy (default "always") decides whether every inbound message triggers a run.
	if inbox.AgentConfigID != nil && *inbox.AgentConfigID != "" {
		policy := string(constants.AgentTriggerPolicyAlways)
		if inbox.AgentTriggerPolicy != nil && *inbox.AgentTriggerPolicy != "" {
			policy = *inbox.AgentTriggerPolicy
		}
		if cErr := seatAgent(*inbox.AgentConfigID, policy, inbox.AgentTriggerKeywords); cErr != nil {
			return "", cErr
		}
	}

	// Seat the inbox's roster (messaging_group): its human members join the case so the team can read, edit,
	// and approve alongside the agent; any agent members are seated too (defaulting to @mention so they don't
	// all auto-run on every inbound — the inbox's own triage agent already covers that). A snapshot: later
	// roster edits don't reach this thread. The group is fetched fresh; a since-deleted group seats nobody.
	if inbox.GroupID != nil && *inbox.GroupID != "" {
		members, apiErr := f.NewMessagingGroupRepo().ListMembers(txCtx, *inbox.GroupID)
		if apiErr != nil {
			return "", apiErr
		}
		seenUsers := map[string]bool{}
		for _, gm := range members {
			switch gm.MemberType {
			case domain.MessagingGroupMemberTypeUser:
				if gm.AccountUserID == nil || *gm.AccountUserID == "" || seenUsers[*gm.AccountUserID] {
					continue
				}
				seenUsers[*gm.AccountUserID] = true
				pid, pgenErr := id.GenID(id.ConversationParticipantIDPrefix, nil)
				if pgenErr != nil {
					return "", pgenErr
				}
				acus := *gm.AccountUserID
				if cErr := partRepo.Create(txCtx, &domain.ConversationParticipant{
					ID:              pid,
					ConversationID:  conversationID,
					AccountID:       inbox.AccountID,
					ParticipantType: string(constants.ParticipantTypeUser),
					AccountUserID:   &acus,
					Role:            string(constants.ParticipantRoleMember),
				}); cErr != nil {
					return "", cErr
				}
			case domain.MessagingGroupMemberTypeAgent:
				if gm.AgentConfigID == nil {
					continue
				}
				if cErr := seatAgent(*gm.AgentConfigID, string(constants.AgentTriggerPolicyMention), nil); cErr != nil {
					return "", cErr
				}
			}
		}
	}
	return conversationID, nil
}

// threadCandidates collects the rfc Message-IDs an inbound mail references (In-Reply-To first, then
// References), de-duplicated, for resolving which conversation it threads onto.
func threadCandidates(inReplyTo string, references []string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	add(inReplyTo)
	for _, r := range references {
		add(r)
	}
	return out
}
