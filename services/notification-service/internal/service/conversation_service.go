package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/services/notification-service/internal/ratelimit"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/audit"
	s3 "github.com/augno/api/shared/cloud/s3"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	"github.com/augno/api/shared/db"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/idempotency"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/ptrutil"
	"github.com/augno/api/shared/tracing"
)

var conversationSvcTracer = tracing.GetTracer("notification-service.conversation_service")

const (
	defaultMessagePageSize int32 = 50
	maxMessagePageSize     int32 = 100
	messagePreviewMaxLen         = 280
)

type conversationSvcImpl struct {
	repoFactory domain.RepoFactory
	txManager   TransactionManager
	objectStore s3.ObjectStore
	chatBucket  string
	// broker publishes ephemeral realtime events (typing) directly to the fanout exchange, bypassing the transactional outbox. May be nil in tests/contexts that never call SendTyping.
	broker messaging.MessageBroker
	// outboxNotifier wakes the outbox enqueuer the instant a message (and any agent-dispatch command it triggers) commits, so the agent run starts without waiting out the enqueuer's idle poll backoff.
	// May be nil in tests/contexts that never post messages.
	outboxNotifier messaging.OutboxNotifier
	// Per-actor anti-abuse rate limiters (§12.10). Generous defaults: a backstop against runaway senders, not a tight quota. Customers get a stricter send bucket than internal staff.
	sendLimiter         *ratelimit.Limiter
	customerSendLimiter *ratelimit.Limiter
	createLimiter       *ratelimit.Limiter
	// bridgeEmailSender sends outbound mail for the email bridge (SES, in the receiving region so the reply comes from the same DKIM-verified identity). May be nil in tests/dev without AWS, in which case SendInboxReply errors out.
	bridgeEmailSender domain.EmailSender
	// inboundEmailDomain is the Augno-owned SES receiving subdomain (e.g. "inbound.augno.com"). When set,
	// inbound routing also resolves mail addressed to <inbox_id>@<this domain> (the per-inbox forwarding
	// address) back to its inbox, so customers who can't repoint their apex MX can forward instead. Empty
	// disables forwarding-address matching (direct customer-MX inboxes still resolve by address).
	inboundEmailDomain string
}

// NewConversationSvc constructs the chat (conversations + messages) service. objectStore and chatBucket back the attachment upload pipeline (presigned PUT/GET against the chat bucket); broker carries ephemeral typing events to the realtime fanout (outside the outbox); outboxNotifier wakes the outbox enqueuer after a message commits so agent dispatch isn't delayed by the idle poll backoff (may be nil); bridgeEmailSender sends outbound email-bridge replies via SES (may be nil where the bridge isn't configured).
func NewConversationSvc(repoFactory domain.RepoFactory, txManager TransactionManager, objectStore s3.ObjectStore, chatBucket string, broker messaging.MessageBroker, outboxNotifier messaging.OutboxNotifier, bridgeEmailSender domain.EmailSender, inboundEmailDomain string) domain.ConversationSvc {
	// Rates are static, known-valid configs, so a construction error is a programming error: fail fast at init.
	sendLimiter, err := ratelimit.New(&ratelimit.Config{Capacity: 300, RefillPerSec: 50})
	if err != nil {
		panic(err)
	}
	customerSendLimiter, err := ratelimit.New(&ratelimit.Config{Capacity: 60, RefillPerSec: 10})
	if err != nil {
		panic(err)
	}
	createLimiter, err := ratelimit.New(&ratelimit.Config{Capacity: 100, RefillPerSec: 20})
	if err != nil {
		panic(err)
	}
	return &conversationSvcImpl{
		repoFactory:         repoFactory,
		txManager:           txManager,
		objectStore:         objectStore,
		chatBucket:          chatBucket,
		broker:              broker,
		outboxNotifier:      outboxNotifier,
		sendLimiter:         sendLimiter,
		customerSendLimiter: customerSendLimiter,
		createLimiter:       createLimiter,
		bridgeEmailSender:   bridgeEmailSender,
		inboundEmailDomain:  inboundEmailDomain,
	}
}

// kickOutbox wakes the outbox enqueuer so a just-committed outbox row (e.g. an agent-dispatch command) is published immediately rather than on the enqueuer's next idle poll. No-op when no notifier was injected. Call only after the writing transaction has committed.
func (s *conversationSvcImpl) kickOutbox() {
	if s.outboxNotifier != nil {
		s.outboxNotifier.Notify()
	}
}

// caller resolves the authenticated identity to its account_user id. Chat requires a real account_user, so api-key actors (no membership) are rejected.
func (s *conversationSvcImpl) caller(ctx context.Context) (*types.Identity, string, string, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || !identity.IsActorSet() {
		return nil, "", "", apierror.NewAuthenticationError("Authentication is required.")
	}
	if !identity.IsTargetAccountSet() {
		return nil, "", "", apierror.NewAuthenticationError("The Augno-Account-ID header is required.")
	}
	accountID := identity.Target.AccountID
	accountUserID, apiErr := s.repoFactory.NewNotificationRepo().ResolveAccountUserID(ctx, identity.Actor.ID, accountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, "", "", apierror.NewAuthorizationError("Chat is only available to users with an account membership.")
		}
		return nil, "", "", apiErr
	}
	return identity, accountUserID, accountID, nil
}

func (s *conversationSvcImpl) CreateConversation(ctx context.Context, input domain.CreateConversationInput) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.create")
	defer span.End()

	_, callerAcus, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Anti-abuse conversation-creation throttle (§12.10), keyed per actor.
	if !s.createLimiter.Allow(callerAcus) {
		return nil, tracing.Trace(span, apierror.NewRateLimitExceededError("You are creating conversations too quickly. Please slow down."))
	}

	switch constants.ConversationType(input.Type) {
	case constants.ConversationTypeDM:
		// fall through to the DM logic below
	case constants.ConversationTypeGroup:
		return s.createGroup(ctx, input, callerAcus, accountID)
	default:
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("Unsupported conversation type.", "type"))
	}

	if len(input.ParticipantAccountUserIDs) != 1 {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("A direct message requires exactly one other participant.", "participant_account_user_ids"))
	}
	target := input.ParticipantAccountUserIDs[0]
	if target == "" {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The target participant is invalid.", "participant_account_user_ids"))
	}
	// A self-DM ("note to self") is an explicitly-allowed conversation where the pair is (caller, caller).
	isSelfDM := target == callerAcus
	// The target must be a real account_user (resolvable to a user id).
	if _, apiErr := s.repoFactory.NewNotificationRepo().ResolveUserID(ctx, target); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The target participant does not exist.", "participant_account_user_ids"))
		}
		return nil, tracing.Trace(span, apiErr)
	}
	// A block in either direction forbids opening a DM (not applicable to a self-DM).
	if !isSelfDM {
		if blocked, apiErr := s.repoFactory.NewBlockRepo().ExistsBetween(ctx, callerAcus, target); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		} else if blocked {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError("You cannot start a conversation with this user."))
		}
	}

	dmKey := buildDMKey(callerAcus, target)

	// Dedup fast path: an existing DM between this pair is returned as-is.
	if existingID, apiErr := s.repoFactory.NewConversationRepo().GetDMConversationID(ctx, accountID, dmKey); apiErr == nil {
		return s.loadConversation(ctx, existingID, callerAcus, accountID)
	} else if apiErr.Code != apierror.ErrorCodeResourceNotFound {
		return nil, tracing.Trace(span, apiErr)
	}

	conversationID, apiErr := id.GenID(id.ConversationIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		convRepo := f.NewConversationRepo()
		if createErr := convRepo.Create(txCtx, conversationID, &input, accountID); createErr != nil {
			return createErr
		}
		if dmErr := convRepo.CreateDMKey(txCtx, accountID, dmKey, conversationID); dmErr != nil {
			if db.IsDuplicateEntry(dmErr) {
				return apierror.NewResourceConflictError("A direct message with this participant already exists.")
			}
			if mapped := db.MapSQLError(dmErr); mapped != nil {
				return mapped
			}
		}
		partRepo := f.NewParticipantRepo()
		// A self-DM has a single participant (the pair is the same account_user).
		participantAcus := []string{callerAcus, target}
		if isSelfDM {
			participantAcus = []string{callerAcus}
		}
		for _, acus := range participantAcus {
			pid, genErr := id.GenID(id.ConversationParticipantIDPrefix, nil)
			if genErr != nil {
				return genErr
			}
			acusCopy := acus
			if pErr := partRepo.Create(txCtx, &domain.ConversationParticipant{
				ID:              pid,
				ConversationID:  conversationID,
				AccountID:       accountID,
				ParticipantType: string(constants.ParticipantTypeUser),
				AccountUserID:   &acusCopy,
				Role:            string(constants.ParticipantRoleMember),
			}); pErr != nil {
				return pErr
			}
		}
		// A genuinely-new DM was persisted (the dedup + race paths return an existing row before reaching here), so publish the create.
		created, getErr := convRepo.GetByID(txCtx, conversationID, accountID)
		if getErr != nil {
			return getErr
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeConversation,
			ResourceID:   conversationID,
			Changes:      audit.ComputeChanges(nil, created),
		})
	})
	if apiErr != nil {
		// A concurrent create raced us to the DM key; return the winner.
		if apiErr.Code == apierror.ErrorCodeResourceConflict {
			if existingID, getErr := s.repoFactory.NewConversationRepo().GetDMConversationID(ctx, accountID, dmKey); getErr == nil {
				return s.loadConversation(ctx, existingID, callerAcus, accountID)
			}
		}
		return nil, tracing.Trace(span, apiErr)
	}

	return s.loadConversation(ctx, conversationID, callerAcus, accountID)
}

// listSupportConversationForRelation returns the single-item conversation list a customer/supplier relation actor sees in the portal: their support thread with the vendor, hydrated like the inbox (participants, unread, last-message preview). Returns an empty page before the customer's first contact, so the portal renders an empty inbox rather than erroring.
func (s *conversationSvcImpl) listSupportConversationForRelation(ctx context.Context, identity *types.Identity, accountID string) (*domain.ConversationPage, *apierror.APIError) {
	customerAccount := identity.ActorAccountID()
	if customerAccount == nil || *customerAccount == "" {
		return &domain.ConversationPage{}, nil
	}
	existing, apiErr := s.repoFactory.NewConversationRepo().GetCustomerSupport(ctx, accountID, *customerAccount)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return &domain.ConversationPage{}, nil
		}
		return nil, apiErr
	}
	// GetConversation is relation-aware (resolves the customer participant) and hydrates participants + unread; add the last-message preview the inbox list shows.
	conv, apiErr := s.GetConversation(ctx, existing.ID)
	if apiErr != nil {
		return nil, apiErr
	}
	// The customer sees only customer-visible messages, so their preview and unread must ignore internal notes — which may well be the conversation's actual last_message.
	lastVisible, apiErr := s.repoFactory.NewMessageRepo().GetLastVisible(ctx, conv.ID)
	if apiErr != nil {
		return nil, apiErr
	}
	conv.LastMessage = lastVisible
	if part, apiErr := s.repoFactory.NewParticipantRepo().GetByRelationAccount(ctx, conv.ID, *customerAccount); apiErr == nil {
		unread, apiErr := s.repoFactory.NewMessageRepo().CountVisibleAfter(ctx, conv.ID, part.LastReadSequence)
		if apiErr != nil {
			return nil, apiErr
		}
		conv.Unread = unread
	}
	return &domain.ConversationPage{Conversations: []*domain.Conversation{conv}}, nil
}

func (s *conversationSvcImpl) ListConversations(ctx context.Context, input domain.ListConversationsInput) (*domain.ConversationPage, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.list")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || !identity.IsActorSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("Authentication is required."))
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}
	accountID := identity.Target.AccountID

	// Customer/supplier relation actors have no account_user and exactly one conversation — their support thread with the vendor. List it (or nothing, before first contact) instead of the staff inbox query.
	if identity.IsRelationActor() {
		return s.listSupportConversationForRelation(ctx, identity, accountID)
	}

	callerAcus, apiErr := s.repoFactory.NewNotificationRepo().ResolveAccountUserID(ctx, identity.Actor.ID, accountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError("Chat is only available to users with an account membership."))
		}
		return nil, tracing.Trace(span, apiErr)
	}

	limit := clampLimit(input.Limit, defaultNotificationPageSize, maxNotificationPageSize)
	cursorAt, cursorID, apiErr := decodeCursor(input.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	status := input.Status
	if status == "" {
		status = string(constants.ConversationListStatusActive)
	}
	listStatus := constants.ConversationListStatus(status)
	if !listStatus.IsValid() {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The status is invalid.", "status"))
	}

	rows, apiErr := s.repoFactory.NewConversationRepo().ListForUser(ctx, domain.ConversationListFilter{
		AccountID:           accountID,
		AccountUserID:       callerAcus,
		Type:                input.Type,
		Status:              status,
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
	// Hydrate participants for the returned page so the inbox UI can render DM/group names + avatars without an N+1 GET per row (which trips the gateway rate limit for large teams). Bounded by the page size.
	partRepo := s.repoFactory.NewParticipantRepo()
	for _, conv := range rows {
		participants, apiErr := partRepo.List(ctx, conv.ID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		conv.Participants = participants
	}

	// Hydrate each conversation's most-recent message for inbox list previews via a single batched fetch keyed on last_message_id — not an N+1 GET per row.
	lastMessageIDs := make([]string, 0, len(rows))
	for _, conv := range rows {
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
				conv.LastMessage = byID[*conv.LastMessageID]
			}
		}
	}
	s.hydrateConversationListPresentation(ctx, rows)

	page.Conversations = rows
	return page, nil
}

// hydrateConversationListPresentation attaches staff-view presentation data to inbox/list rows: customer participant contact names and last-message sender names (cross-account ids the gateway cannot hydrate).
func (s *conversationSvcImpl) hydrateConversationListPresentation(ctx context.Context, rows []*domain.Conversation) {
	for _, conv := range rows {
		s.hydrateCustomerParticipantContacts(ctx, conv.Participants)
		if conv.LastMessage != nil {
			s.resolveSenders(ctx, conv.ID, conv.AccountID, []*domain.Message{conv.LastMessage}, false)
		}
	}
}

func (s *conversationSvcImpl) GetConversation(ctx context.Context, conversationID string) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.get")
	defer span.End()

	// allowLeft: a member who just left can still read the conversation back (it renders as hidden).
	rc, apiErr := s.resolveParticipant(ctx, conversationID, true)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, rc.accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	participants, apiErr := s.repoFactory.NewParticipantRepo().List(ctx, conversationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	conv.Participants = participants
	conv.Unread = unreadFrom(conv.NextSequence, rc.part.LastReadSequence)
	conv.Hidden = rc.part.HiddenAt != nil
	if !rc.isCustomer {
		s.hydrateCustomerParticipantContacts(ctx, participants)
	}
	// Hydrate the most-recent message so ?include=last_message works on the detail endpoint identically to the list path (which batch-loads it above).
	if conv.LastMessageID != nil && *conv.LastMessageID != "" {
		msgs, apiErr := s.repoFactory.NewMessageRepo().GetByIDs(ctx, []string{*conv.LastMessageID})
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, m := range msgs {
			if m.ID == *conv.LastMessageID {
				conv.LastMessage = m
				break
			}
		}
	}
	// A customer viewing their portal support case sees the vendor side as a single branded "Customer Service" party, never the individual staff seated behind it (or an empty "Unknown" when none are). Per-view only — staff viewing the same thread still derive the customer's name.
	if rc.isCustomer && conv.Audience == string(constants.ConversationAudienceCustomer) && conversationHasCustomerParticipant(participants) {
		title := "Customer Service"
		conv.Title = &title
		// The customer is oblivious to who — and how many — staff sit behind "Customer Service". Strip every
		// vendor-side participant so the roster (and its length, which would otherwise leak the head count) never
		// reaches the customer; only the customer's own participant rows remain. Mirrors the author anonymization
		// in resolveSenders. GroupID is provenance only, but keeping it would let a customer expand the seeding
		// group's member list via ?include=group, re-leaking the count — null it for this view.
		conv.Participants = filterCustomerVisibleParticipants(participants)
		conv.GroupID = nil
	}
	return conv, nil
}

// BatchGetConversations resolves each requested conversation through the same participant gate as GetConversation, omitting any the caller cannot access. Used for ?include= expansion at the gateway.
func (s *conversationSvcImpl) BatchGetConversations(ctx context.Context, ids []string) ([]*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.batch_get")
	defer span.End()

	out := make([]*domain.Conversation, 0, len(ids))
	for _, convID := range ids {
		if convID == "" {
			continue
		}
		conv, apiErr := s.GetConversation(ctx, convID)
		if apiErr != nil {
			// Not a member / not found → omit. Surface only unexpected (non-404) failures.
			if apiErr.Code == apierror.ErrorCodeResourceNotFound {
				continue
			}
			return nil, tracing.Trace(span, apiErr)
		}
		out = append(out, conv)
	}
	return out, nil
}

// BatchGetMessages loads messages by id, then drops any whose conversation the caller does not participate in, enriching the survivors with sender attribution + attachments exactly as ListMessages does. Used for ?include= expansion (e.g. a message's reply_to) at the gateway.
func (s *conversationSvcImpl) BatchGetMessages(ctx context.Context, ids []string) ([]*domain.Message, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.batch_get_messages")
	defer span.End()

	if len(ids) == 0 {
		return nil, nil
	}
	rows, apiErr := s.repoFactory.NewMessageRepo().GetByIDs(ctx, ids)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	byConv := make(map[string][]*domain.Message)
	for _, m := range rows {
		byConv[m.ConversationID] = append(byConv[m.ConversationID], m)
	}
	out := make([]*domain.Message, 0, len(rows))
	for convID, msgs := range byConv {
		rc, apiErr := s.resolveParticipant(ctx, convID, false)
		if apiErr != nil {
			// Caller is not an active participant of this conversation → omit its messages.
			continue
		}
		s.resolveSenders(ctx, convID, rc.accountID, msgs, rc.identity.IsRelationActor())
		s.resolveAttachments(ctx, msgs)
		out = append(out, msgs...)
	}
	return out, nil
}

// ContactSupport returns (creating on first contact) the calling customer's portal support case in the vendor account. Only customer relation actors may use it.
func (s *conversationSvcImpl) ContactSupport(ctx context.Context) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.contact_support")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || !identity.IsActorSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("Authentication is required."))
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}
	if !identity.IsRelationActor() {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("Only customer accounts can contact support."))
	}
	vendorAccountID := identity.Target.AccountID
	customerAccount := identity.ActorAccountID()
	if customerAccount == nil || *customerAccount == "" {
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Customer account could not be resolved."))
	}
	// The account_user, in the customer's own account, of the person opening the case. It becomes the
	// customer participant's user actor so staff see who reached out (the customer company itself comes
	// from the case topic). Best-effort: a resolution miss leaves the participant actorless rather than
	// blocking support.
	senderAcus, resolveErr := s.repoFactory.NewNotificationRepo().ResolveAccountUserID(ctx, identity.Actor.ID, *customerAccount)
	if resolveErr != nil {
		slog.WarnContext(ctx, "failed to resolve customer contact account_user; opening case without a contact actor", "error", resolveErr.Error(), "customer_account_id", *customerAccount)
	}

	convRepo := s.repoFactory.NewConversationRepo()
	if existing, apiErr := convRepo.GetCustomerSupport(ctx, vendorAccountID, *customerAccount); apiErr == nil {
		return s.GetConversation(ctx, existing.ID)
	} else if apiErr.Code != apierror.ErrorCodeResourceNotFound {
		return nil, tracing.Trace(span, apiErr)
	}

	// Resolve the designated support recipients (the configured support route's group-conversation participants) before creating anything. Support is only offered when the vendor has set up a route with at least one recipient — otherwise a customer would be messaging into a void, so we refuse to open the thread (the portal also hides the feature via SupportAvailability). Existing threads were already returned above, so this only gates first contact.
	contactRecipients := s.resolveSupportContact(ctx, vendorAccountID, *customerAccount)
	if len(contactRecipients) == 0 {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Support is not available for this account."))
	}

	conversationID, apiErr := id.GenID(id.ConversationIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if cErr := f.NewConversationRepo().CreateCustomerSupport(txCtx, conversationID, vendorAccountID, *customerAccount); cErr != nil {
			return cErr
		}
		pid, genErr := id.GenID(id.ConversationParticipantIDPrefix, nil)
		if genErr != nil {
			return genErr
		}
		if cErr := f.NewParticipantRepo().CreateCustomer(txCtx, pid, conversationID, vendorAccountID, *customerAccount, senderAcus); cErr != nil {
			return cErr
		}
		// Seat each resolved contact as a participant so the inbound message has a deterministic recipient that fanoutMessageNotifications will notify. Augments — does not replace — open lazy-join.
		for _, acus := range contactRecipients {
			seatID, genErr := id.GenID(id.ConversationParticipantIDPrefix, nil)
			if genErr != nil {
				return genErr
			}
			acusCopy := acus
			if cErr := f.NewParticipantRepo().Create(txCtx, &domain.ConversationParticipant{
				ID:              seatID,
				ConversationID:  conversationID,
				AccountID:       vendorAccountID,
				ParticipantType: string(constants.ParticipantTypeUser),
				AccountUserID:   &acusCopy,
				Role:            string(constants.ParticipantRoleMember),
			}); cErr != nil {
				return cErr
			}
		}
		return nil
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.GetConversation(ctx, conversationID)
}

// SupportAvailability reports whether the calling customer can contact support: true only when the vendor has configured a support route that resolves to at least one recipient. The portal gates the contact-support feature on this so customers never message into a void.
func (s *conversationSvcImpl) SupportAvailability(ctx context.Context) (bool, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.support_availability")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || !identity.IsActorSet() {
		return false, tracing.Trace(span, apierror.NewAuthenticationError("Authentication is required."))
	}
	if !identity.IsTargetAccountSet() {
		return false, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}
	if !identity.IsRelationActor() {
		return false, tracing.Trace(span, apierror.NewAuthorizationError("Only customer accounts can check support availability."))
	}
	customerAccount := identity.ActorAccountID()
	if customerAccount == nil || *customerAccount == "" {
		return false, tracing.Trace(span, apierror.NewInvariantViolationError("Customer account could not be resolved."))
	}
	return len(s.resolveSupportContact(ctx, identity.Target.AccountID, *customerAccount)) > 0, nil
}

// resolveSupportContact resolves the group conversation designated to handle this vendor↔customer relationship's support (a per-relation override winning over the account default) and returns its active human participants' account_user ids to seat. Best-effort: no configured route or any resolution error yields no recipients rather than failing the customer's request, so support remains reachable via the existing open lazy-join path.
func (s *conversationSvcImpl) resolveSupportContact(ctx context.Context, vendorAccountID, customerAccountID string) []string {
	return resolveSupportRecipients(ctx, s.repoFactory, vendorAccountID, customerAccountID)
}

// resolveSupportRecipients resolves the support-route group designated to handle this vendor↔customer relationship (a per-relation override winning over the account default) and returns its active human participants' account_user ids. Best-effort: no configured route or any resolution error yields no recipients rather than an error, so callers degrade gracefully.
func resolveSupportRecipients(ctx context.Context, repoFactory domain.RepoFactory, vendorAccountID, customerAccountID string) []string {
	route, apiErr := repoFactory.NewSupportRouteRepo().Resolve(ctx, vendorAccountID, customerAccountID)
	if apiErr != nil {
		slog.WarnContext(ctx, "failed to resolve support route", "error", apiErr.Error(), "vendor_account_id", vendorAccountID)
		return nil
	}
	if route == nil {
		return nil
	}
	participants, apiErr := repoFactory.NewParticipantRepo().List(ctx, route.GroupConversationID)
	if apiErr != nil {
		slog.WarnContext(ctx, "failed to list support group participants", "error", apiErr.Error(), "group_conversation_id", route.GroupConversationID)
		return nil
	}
	var recipients []string
	for _, p := range participants {
		if p.ParticipantType == string(constants.ParticipantTypeUser) && p.AccountUserID != nil && *p.AccountUserID != "" {
			recipients = append(recipients, *p.AccountUserID)
		}
	}
	return recipients
}

// SetSupportRoute designates (or re-points) the group conversation that handles support for a scope.
// relationAccountID is the scope: "" = the account default for any customer; a concrete account id is a per-relation override. The target must be a group conversation owned by the caller's account.
func (s *conversationSvcImpl) SetSupportRoute(ctx context.Context, relationAccountID, groupConversationID string) (*domain.SupportRoute, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.set_support_route")
	defer span.End()

	accountID, apiErr := s.requireMessagingAdmin(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if groupConversationID == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("A group_conversation_id is required.", "group_conversation_id"))
	}
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, groupConversationID, accountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The group conversation does not exist in this account.", "group_conversation_id"))
		}
		return nil, tracing.Trace(span, apiErr)
	}
	if conv.Type != string(constants.ConversationTypeGroup) {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The support route target must be a group conversation.", "group_conversation_id"))
	}

	routeID, apiErr := id.GenID(id.SupportRouteIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Whether a route already exists for this scope decides create vs update on the upsert.
	prior, apiErr := s.repoFactory.NewSupportRouteRepo().Get(ctx, accountID, relationAccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		routeRepo := f.NewSupportRouteRepo()
		if apiErr := routeRepo.Upsert(txCtx, routeID, accountID, relationAccountID, groupConversationID); apiErr != nil {
			return apiErr
		}
		route, apiErr := routeRepo.Get(txCtx, accountID, relationAccountID)
		if apiErr != nil {
			return apiErr
		}
		if route == nil {
			return apierror.NewInvariantViolationError("Support route not found after upsert.")
		}
		action := constants.AuditActionCreate
		changes := audit.ComputeChanges(nil, route)
		if prior != nil {
			action = constants.AuditActionUpdate
			changes = audit.ComputeChanges(prior, route)
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       action,
			ResourceType: constants.ObjectTypeSupportRoute,
			ResourceID:   route.ID,
			Changes:      changes,
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewSupportRouteRepo().Get(ctx, accountID, relationAccountID)
}

// ClearSupportRoute removes the route for a scope in the caller's account.
func (s *conversationSvcImpl) ClearSupportRoute(ctx context.Context, relationAccountID string) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.clear_support_route")
	defer span.End()

	accountID, apiErr := s.requireMessagingAdmin(ctx, types.ActionUpdate)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	// Load the pre-image for the delete event's resource id + field diff. No route → nothing to delete or audit.
	existing, apiErr := s.repoFactory.NewSupportRouteRepo().Get(ctx, accountID, relationAccountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if existing == nil {
		return tracing.Trace(span, apierror.NewResourceNotFoundError("Support route not found."))
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		deleted, apiErr := f.NewSupportRouteRepo().Delete(txCtx, accountID, relationAccountID)
		if apiErr != nil {
			return apiErr
		}
		if !deleted {
			return apierror.NewResourceNotFoundError("Support route not found.")
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeSupportRoute,
			ResourceID:   existing.ID,
			Changes:      audit.ComputeChanges(existing, (*domain.SupportRoute)(nil)),
		})
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

// GetSupportRoute returns the route for an exact scope in the caller's account.
func (s *conversationSvcImpl) GetSupportRoute(ctx context.Context, relationAccountID string) (*domain.SupportRoute, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.get_support_route")
	defer span.End()

	accountID, apiErr := s.requireMessagingAdmin(ctx, types.ActionRead)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	route, apiErr := s.repoFactory.NewSupportRouteRepo().Get(ctx, accountID, relationAccountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if route == nil {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Support route not found."))
	}
	return route, nil
}

// resolvedCaller is the caller's effective participant context. Internal actors resolve to an account_user participant; customer/supplier relation actors resolve to a customer participant keyed by their account. The conversation/message gates work uniformly off the resolved participant row.
type resolvedCaller struct {
	identity   *types.Identity
	accountID  string // the vendor (target) account that owns the conversation
	part       *domain.ConversationParticipant
	isCustomer bool
	// senderAcus is the account_user id for internal callers, or "" for customer callers.
	senderAcus string
}

// resolveParticipant resolves the caller's active participant row in a conversation, supporting both internal account_user actors and customer relation actors. Not-found / inactive yields a 404 (the conversation's existence is never leaked to a non-member).
// resolveParticipant gates access to a conversation for the caller. When allowLeft is true, a
// participant who voluntarily left (membership='left') is still permitted — used by the read path so
// a member can fetch back the conversation immediately after leaving. Removed ('removed') participants
// are never permitted.
func (s *conversationSvcImpl) resolveParticipant(ctx context.Context, conversationID string, allowLeft bool) (*resolvedCaller, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || !identity.IsActorSet() {
		return nil, apierror.NewAuthenticationError("Authentication is required.")
	}
	if !identity.IsTargetAccountSet() {
		return nil, apierror.NewAuthenticationError("The Augno-Account-ID header is required.")
	}
	accountID := identity.Target.AccountID

	if identity.IsRelationActor() {
		customerAccount := identity.ActorAccountID()
		if customerAccount == nil || *customerAccount == "" {
			return nil, apierror.NewResourceNotFoundError("Conversation not found.")
		}
		part, apiErr := s.repoFactory.NewParticipantRepo().GetByRelationAccount(ctx, conversationID, *customerAccount)
		if apiErr != nil {
			if apiErr.Code == apierror.ErrorCodeResourceNotFound {
				return nil, apierror.NewResourceNotFoundError("Conversation not found.")
			}
			return nil, apiErr
		}
		if part.Membership != string(constants.ParticipantMembershipActive) {
			return nil, apierror.NewResourceNotFoundError("Conversation not found.")
		}
		return &resolvedCaller{identity: identity, accountID: accountID, part: part, isCustomer: true}, nil
	}

	callerAcus, apiErr := s.repoFactory.NewNotificationRepo().ResolveAccountUserID(ctx, identity.Actor.ID, accountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apierror.NewAuthorizationError("Chat is only available to users with an account membership.")
		}
		return nil, apiErr
	}
	part, apiErr := s.repoFactory.NewParticipantRepo().Get(ctx, conversationID, callerAcus)
	if apiErr != nil {
		if apiErr.Code != apierror.ErrorCodeResourceNotFound {
			return nil, apiErr
		}
		// Internal staff with messaging permission may access a customer-audience case they have not joined yet; they lazy-join as a participant on first interaction so their replies carry a sender participant.
		joined, joinErr := s.lazyJoinSupportStaff(ctx, conversationID, accountID, callerAcus, identity)
		if joinErr != nil {
			return nil, joinErr
		}
		if joined == nil {
			return nil, apierror.NewResourceNotFoundError("Conversation not found.")
		}
		part = joined
	}
	if part.Membership != string(constants.ParticipantMembershipActive) &&
		!(allowLeft && part.Membership == string(constants.ParticipantMembershipLeft)) {
		return nil, apierror.NewResourceNotFoundError("Conversation not found.")
	}
	return &resolvedCaller{identity: identity, accountID: accountID, part: part, senderAcus: callerAcus}, nil
}

// lazyJoinSupportStaff adds an internal actor as a participant of a customer-audience case if they hold the messaging permission, returning the new participant. Returns (nil, nil) when the conversation isn't customer-facing or the actor lacks permission (caller treats as not-found).
func (s *conversationSvcImpl) lazyJoinSupportStaff(ctx context.Context, conversationID, accountID, callerAcus string, identity *types.Identity) (*domain.ConversationParticipant, *apierror.APIError) {
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, accountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil
		}
		return nil, apiErr
	}
	if conv.Audience != string(constants.ConversationAudienceCustomer) {
		return nil, nil
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMessaging, types.ActionRead); apiErr != nil {
		return nil, nil // not a support agent — no access
	}
	pid, apiErr := id.GenID(id.ConversationParticipantIDPrefix, nil)
	if apiErr != nil {
		return nil, apiErr
	}
	if apiErr := s.repoFactory.NewParticipantRepo().Create(ctx, &domain.ConversationParticipant{
		ID:              pid,
		ConversationID:  conversationID,
		AccountID:       accountID,
		ParticipantType: string(constants.ParticipantTypeUser),
		AccountUserID:   &callerAcus,
		Role:            string(constants.ParticipantRoleMember),
	}); apiErr != nil {
		return nil, apiErr
	}
	return s.repoFactory.NewParticipantRepo().Get(ctx, conversationID, callerAcus)
}

// resolveContactDisplayName resolves an account_user's display name via the notification DB (cross-account safe).
func (s *conversationSvcImpl) resolveContactDisplayName(ctx context.Context, accountUserID string) string {
	if accountUserID == "" {
		return ""
	}
	contact, apiErr := s.repoFactory.NewNotificationRepo().ResolveRecipientContact(ctx, accountUserID)
	if apiErr != nil || contact == nil {
		return ""
	}
	return contact.Name
}

// hydrateCustomerParticipantContacts resolves the contact display name on customer participants so staff see who opened the case. The account_user lives in the customer's account, so the gateway's vendor-scoped account_user batch loader cannot hydrate it.
func (s *conversationSvcImpl) hydrateCustomerParticipantContacts(ctx context.Context, participants []*domain.ConversationParticipant) {
	nameCache := make(map[string]string)
	for _, p := range participants {
		if p.ParticipantType != string(constants.ParticipantTypeCustomer) || p.AccountUserID == nil || *p.AccountUserID == "" {
			continue
		}
		acus := *p.AccountUserID
		name, ok := nameCache[acus]
		if !ok {
			name = s.resolveContactDisplayName(ctx, acus)
			nameCache[acus] = name
		}
		if name != "" {
			p.AccountUserDisplayName = &name
		}
	}
}

func conversationHasCustomerParticipant(participants []*domain.ConversationParticipant) bool {
	for _, p := range participants {
		if p.ParticipantType == string(constants.ParticipantTypeCustomer) {
			return true
		}
	}
	return false
}

// filterCustomerVisibleParticipants keeps only the customer-side participants, dropping every vendor-side
// (staff/agent) row. A customer viewing their support case must not see who — or how many — people are behind
// the single "Customer Service" party, and the participant array's length would otherwise leak that count.
func filterCustomerVisibleParticipants(participants []*domain.ConversationParticipant) []*domain.ConversationParticipant {
	out := make([]*domain.ConversationParticipant, 0, len(participants))
	for _, p := range participants {
		if p.ParticipantType == string(constants.ParticipantTypeCustomer) {
			out = append(out, p)
		}
	}
	return out
}

// loadConversation returns a conversation the caller participates in, with participants and the caller's computed unread. Non-participants get a not-found (existence is not leaked).
func (s *conversationSvcImpl) loadConversation(ctx context.Context, conversationID, callerAcus, accountID string) (*domain.Conversation, *apierror.APIError) {
	part, apiErr := s.repoFactory.NewParticipantRepo().Get(ctx, conversationID, callerAcus)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apierror.NewResourceNotFoundError("Conversation not found.")
		}
		return nil, apiErr
	}
	if part.Membership != string(constants.ParticipantMembershipActive) {
		return nil, apierror.NewResourceNotFoundError("Conversation not found.") // left/removed: no access
	}

	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, accountID)
	if apiErr != nil {
		return nil, apiErr
	}
	participants, apiErr := s.repoFactory.NewParticipantRepo().List(ctx, conversationID)
	if apiErr != nil {
		return nil, apiErr
	}
	conv.Participants = participants
	conv.Unread = unreadFrom(conv.NextSequence, part.LastReadSequence)
	conv.Hidden = part.HiddenAt != nil
	return conv, nil
}

// resolveSendVisibility decides the persisted visibility of a new message against the conversation's audience and who is sending. This is the server-side safety gate behind the two send paths.
//
//   - Internal conversation (audience != customer): everything is forced to "internal". Requesting
//     "external"/"system" here is a client error (there is no external party to address).
//   - External case (audience == customer):
//   - the customer themselves always posts "external" (their inbound message is part of the
//     official customer history);
//   - staff/agents default to "internal" (a team note) when unspecified; an explicit "external"
//     (the reply-customer path, permission-gated at the gateway) or "system" is honored.
func resolveSendVisibility(audience string, isCustomer bool, requested string) (string, *apierror.APIError) {
	requested = strings.TrimSpace(requested)
	if requested != "" && !constants.MessageVisibility(requested).IsValid() {
		return "", apierror.NewParameterInvalidError("Unknown message visibility.", "visibility")
	}

	if audience != string(constants.ConversationAudienceCustomer) {
		if requested == string(constants.MessageVisibilityExternal) || requested == string(constants.MessageVisibilitySystem) {
			return "", apierror.NewParameterInvalidError("This conversation has no customer; only internal messages are allowed.", "visibility")
		}
		return string(constants.MessageVisibilityInternal), nil
	}

	if isCustomer {
		return string(constants.MessageVisibilityExternal), nil
	}
	if requested == "" {
		return string(constants.MessageVisibilityInternal), nil
	}
	return requested, nil
}

func (s *conversationSvcImpl) SendMessage(ctx context.Context, input domain.SendMessageInput) (*domain.Message, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.send_message")
	defer span.End()

	rc, apiErr := s.resolveParticipant(ctx, input.ConversationID, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	identity, part, accountID, callerAcus := rc.identity, rc.part, rc.accountID, rc.senderAcus

	// Anti-abuse send throttle (§12.10): customers get a stricter bucket; keyed per actor.
	sendLimiter, rateKey := s.sendLimiter, callerAcus
	if rc.isCustomer {
		sendLimiter, rateKey = s.customerSendLimiter, part.ID
	}
	if !sendLimiter.Allow(rateKey) {
		return nil, tracing.Trace(span, apierror.NewRateLimitExceededError("You are sending messages too quickly. Please slow down."))
	}

	// A message needs text, at least one attachment, or a resource link.
	hasLink := input.LinkResourceID != nil && *input.LinkResourceID != ""
	if (input.Body == nil || *input.Body == "") && len(input.Attachments) == 0 && !hasLink {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("A message body, attachment, or link is required.", "body"))
	}
	if input.ClientMessageID == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("A client_message_id is required.", "client_message_id"))
	}
	if part.Role == string(constants.ParticipantRoleViewer) {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You cannot post to this conversation."))
	}

	if !rc.isCustomer {
		// In a DM, a block in either direction (possibly added after the DM was created) forbids sending.
		if apiErr := s.enforceDMBlock(ctx, input.ConversationID, accountID, callerAcus); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Resolve the message's visibility against the conversation's audience. This is the safety gate that keeps an internal note out of the customer-visible timeline (and vice versa).
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, input.ConversationID, accountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Conversation not found."))
		}
		return nil, tracing.Trace(span, apiErr)
	}
	visibility, apiErr := resolveSendVisibility(conv.Audience, rc.isCustomer, input.Visibility)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Idempotency fast path: a resend with the same client_message_id returns the original.
	if existing, apiErr := s.repoFactory.NewMessageRepo().GetByClientID(ctx, input.ConversationID, input.ClientMessageID); apiErr == nil {
		s.resolveSenders(ctx, input.ConversationID, accountID, []*domain.Message{existing}, identity.IsRelationActor())
		return existing, nil
	} else if apiErr.Code != apierror.ErrorCodeResourceNotFound {
		return nil, tracing.Trace(span, apiErr)
	}

	// Staff customer-visible reply on an external case (audience=customer): the folded-in reply-customer
	// path. The customer always sees it branded "Customer Service" (applied at read time, see
	// resolveSenders), and an email-bridged case is delivered as outbound mail rather than a portal
	// message. After it lands, the ball is back in the customer's court (workflow -> waiting_external).
	isCustomerReply := visibility == string(constants.MessageVisibilityExternal) && !rc.isCustomer
	if isCustomerReply && conv.EmailInboxID != nil && *conv.EmailInboxID != "" {
		subject := caseEmailSubject(input.Subject, conv.Title)
		sent, apiErr := s.deliverInboxReply(ctx, input.ConversationID, part, "", "", ptrutil.Deref(input.Body), subject, input.Cc)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		s.autoSetCaseWorkflow(ctx, conv.ID, accountID, constants.ConversationWorkflowStatusWaitingExternal)
		return sent, nil
	}
	// A portal customer-reply falls through to the normal create path; the read-time collapse brands it.

	messageID, apiErr := id.GenID(id.MessageIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Validate + build attachment rows before the transaction (object existence is a foreign read).
	attachments, apiErr := s.buildAttachments(ctx, input.ConversationID, accountID, messageID, input.Attachments)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	var result *domain.Message
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		convRepo := f.NewConversationRepo()
		seq, lockErr := convRepo.LockSequence(txCtx, input.ConversationID, accountID)
		if lockErr != nil {
			if lockErr.Code == apierror.ErrorCodeResourceNotFound {
				return apierror.NewResourceNotFoundError("Conversation not found.")
			}
			return lockErr
		}

		now := time.Now()
		clientMsgID := input.ClientMessageID
		msg := &domain.Message{
			ID:                  messageID,
			ConversationID:      input.ConversationID,
			AccountID:           accountID,
			Sequence:            seq,
			Kind:                string(constants.MessageKindChat),
			Visibility:          visibility,
			Channel:             new(string(constants.MessageChannelMessage)),
			SenderParticipantID: &part.ID,
			ClientMessageID:     &clientMsgID,
			Body:                input.Body,
			Preview:             strPtrIfNotEmpty(messagePreview(input.Body, len(input.Attachments), hasLink)),
			LinkResourceType:    input.LinkResourceType,
			LinkResourceID:      input.LinkResourceID,
			ReplyToMessageID:    input.ReplyToMessageID,
			CreatedAt:           now,

			MentionAccountUserIDs: input.MentionAccountUserIDs,
		}
		inserted, createErr := f.NewMessageRepo().Create(txCtx, msg)
		if createErr != nil {
			return createErr
		}
		if !inserted {
			// A concurrent resend won; return the persisted winner without re-advancing.
			existing, getErr := f.NewMessageRepo().GetByClientID(txCtx, input.ConversationID, input.ClientMessageID)
			if getErr != nil {
				return getErr
			}
			result = existing
			return nil
		}

		// Persist attachments (validated above) within the same transaction as the message.
		attachmentRepo := f.NewMessageAttachmentRepo()
		for _, a := range attachments {
			if attErr := attachmentRepo.Create(txCtx, a); attErr != nil {
				return attErr
			}
		}
		msg.Attachments = attachments

		if advErr := convRepo.AdvanceAfterMessage(txCtx, input.ConversationID, messageID, now); advErr != nil {
			return advErr
		}
		// The author has implicitly read their own message (cursor advance by participant id works for both internal account_user and customer participants).
		if curErr := f.NewParticipantRepo().AdvanceReadCursorByID(txCtx, part.ID, messageID, seq); curErr != nil {
			return curErr
		}

		if rtErr := s.fanoutMessageRealtime(txCtx, f, input.ConversationID, accountID, messageID, seq, visibility); rtErr != nil {
			return rtErr
		}

		// Bell feed + email bridge: per-recipient notification rows honoring preferences and mute.
		if bellErr := s.fanoutMessageNotifications(txCtx, f, msg, accountID, callerAcus); bellErr != nil {
			return bellErr
		}

		// Agent dispatch: evaluate agent participants and (for run-linked conversations) enqueue an agent continuation when a trigger policy matches.
		if dispErr := s.dispatchAgents(txCtx, f, input.ConversationID, accountID, part.ParticipantType, callerAcus, msg); dispErr != nil {
			return dispErr
		}

		result = msg
		return nil
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// The message and any agent-dispatch command it enqueued are committed — wake the enqueuer so the agent run starts (and its live "thinking" indicator appears) immediately rather than after an idle poll backoff. Also shortens realtime delivery of the message itself.
	s.kickOutbox()

	s.resolveSenders(ctx, input.ConversationID, accountID, []*domain.Message{result}, identity.IsRelationActor())
	s.resolveAttachments(ctx, []*domain.Message{result})

	// A customer's own inbound message on an external case puts the ball back in the team's court. (No-op for internal conversations and staff/agent sends — only a customer participant triggers it.)
	if rc.isCustomer {
		s.advanceCaseOnCustomerInbound(ctx, input.ConversationID, accountID)
	}
	// A staff portal reply to the customer leaves the ball in the customer's court.
	if isCustomerReply {
		s.autoSetCaseWorkflow(ctx, input.ConversationID, accountID, constants.ConversationWorkflowStatusWaitingExternal)
	}
	return result, nil
}

// fanoutMessageRealtime emits a message.created event per active user participant. Each event targets the recipient's user topic (conversation list / unread) and the conversation topic (live thread); clients dedupe by message id. visibility is carried so a customer-subscribed socket can drop internal frames (see RealtimeDeliveryData.Visibility).
func (s *conversationSvcImpl) fanoutMessageRealtime(ctx context.Context, f domain.RepoFactory, conversationID, accountID, messageID string, sequence int64, visibility string) *apierror.APIError {
	participants, apiErr := f.NewParticipantRepo().List(ctx, conversationID)
	if apiErr != nil {
		return apiErr
	}
	notifRepo := f.NewNotificationRepo()
	outbox := f.NewOutboxRepo()
	for _, p := range participants {
		if p.AccountUserID == nil || *p.AccountUserID == "" {
			continue
		}
		userID, apiErr := notifRepo.ResolveUserID(ctx, *p.AccountUserID)
		if apiErr != nil {
			continue // unknown account_user — skip the push (persisted message still reconciles)
		}
		if rtErr := enqueueRealtimeVia(ctx, outbox, messaging.RealtimeDeliveryData{
			AccountID:       accountID,
			RecipientUserID: userID,
			ConversationID:  conversationID,
			Event:           "message.created",
			Visibility:      visibility,
			MessageID:       messageID,
			Sequence:        sequence,
		}); rtErr != nil {
			return rtErr
		}
	}
	return nil
}

// fanoutMessageNotifications writes a bell-feed notification row per eligible recipient and, when the recipient's email channel is enabled with an instant digest, enqueues a chat-notification email. Recipients who muted the conversation are skipped entirely; the in-app channel is gated by the recipient's notification_preference for the chat.message category (default: on).
func (s *conversationSvcImpl) fanoutMessageNotifications(ctx context.Context, f domain.RepoFactory, msg *domain.Message, accountID, senderAcus string) *apierror.APIError {
	participants, apiErr := f.NewParticipantRepo().List(ctx, msg.ConversationID)
	if apiErr != nil {
		return apiErr
	}

	notifRepo := f.NewNotificationRepo()
	prefRepo := f.NewNotificationPreferenceRepo()
	outbox := f.NewOutboxRepo()

	senderType, senderID, senderName := s.messageSenderAttribution(ctx, f, msg, senderAcus)
	var conversationTitle string
	var isCustomerCase bool
	if conv, apiErr := f.NewConversationRepo().GetByID(ctx, msg.ConversationID, accountID); apiErr == nil {
		if conv.Title != nil {
			conversationTitle = *conv.Title
		}
		// Customer-facing cases live in the support inbox, not team messages; tag their notifications so the
		// client routes the link there (see the notification resource link built by the gateway).
		isCustomerCase = conv.Audience == string(constants.ConversationAudienceCustomer)
	}

	title := "New message"
	if senderName != "" {
		title = senderName
	}

	mentioned := make(map[string]bool, len(msg.MentionAccountUserIDs))
	for _, m := range msg.MentionAccountUserIDs {
		mentioned[m] = true
	}

	for _, p := range participants {
		if p.AccountUserID == nil || *p.AccountUserID == "" || *p.AccountUserID == senderAcus {
			continue
		}
		if p.Membership != string(constants.ParticipantMembershipActive) {
			continue
		}
		isMention := mentioned[*p.AccountUserID]
		// Mute suppresses the bell row and the email; the conversation still counts as unread. A direct @mention is the one exception: it always writes a bell (but never an email when muted).
		if p.Notifications == string(constants.ParticipantNotificationsMuted) && !isMention {
			continue
		}

		category := string(constants.NotificationCategoryChatMessage)
		if isMention {
			category = string(constants.NotificationCategoryChatMention)
		}
		eff := s.effectivePreference(ctx, prefRepo, *p.AccountUserID, category)

		// A mention pierces both mute and the in-app preference so the recipient is always alerted.
		if eff.InAppEnabled || isMention {
			notifID, apiErr := id.GenID(id.NotificationIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			// A direct @mention is a dedicated, high-priority alert (stands out in the bell) — even when the conversation is muted.
			priority := constants.NotificationPriorityNormal
			notifTitle := title
			if isMention {
				priority = constants.NotificationPriorityHigh
				if senderName != "" {
					notifTitle = senderName + " mentioned you"
				} else {
					notifTitle = "You were mentioned"
				}
			}
			n := &domain.Notification{
				ID:                     notifID,
				AccountID:              accountID,
				RecipientAccountUserID: *p.AccountUserID,
				Category:               category,
				SourceMessageID:        &msg.ID,
				ConversationID:         &msg.ConversationID,
				Title:                  notifTitle,
				Body:                   msg.Preview,
				SenderType:             strPtrIfNotEmpty(senderType),
				SenderID:               strPtrIfNotEmpty(senderID),
				SenderName:             strPtrIfNotEmpty(senderName),
				Priority:               string(priority),
			}
			if isCustomerCase {
				supportCaseType := string(constants.ObjectTypeSupportCase)
				n.LinkResourceType = &supportCaseType
				n.LinkResourceID = &msg.ConversationID
			}
			if apiErr := notifRepo.Create(ctx, n); apiErr != nil {
				return apiErr
			}
			// Push the mention as a dedicated bell event to the recipient's user topic, so it alerts live even when the conversation is muted (muted chats suppress the per-message push).
			if isMention {
				if userID, uidErr := notifRepo.ResolveUserID(ctx, *p.AccountUserID); uidErr == nil {
					_ = enqueueRealtimeVia(ctx, outbox, messaging.RealtimeDeliveryData{
						AccountID:              accountID,
						RecipientUserID:        userID,
						RecipientAccountUserID: *p.AccountUserID,
						ConversationID:         msg.ConversationID,
						Event:                  "notification.created",
						NotificationID:         n.ID,
					})
				}
			}
		}

		// Email follows preferences and never fires for a muted recipient (even when mentioned).
		if p.Notifications != string(constants.ParticipantNotificationsMuted) && eff.EmailEnabled && eff.Digest == string(constants.NotificationDigestInstant) {
			if apiErr := s.enqueueChatEmail(ctx, f, outbox, *p.AccountUserID, accountID, senderName, conversationTitle, msg.Preview); apiErr != nil {
				return apiErr
			}
		}
	}
	return nil
}

// messageSenderAttribution resolves the (type, id, name) shown as the sender of a notification: an agent author, else the authoring account user. (In-app notifications go only to internal recipients, so no customer-facing anonymization applies here.)
func (s *conversationSvcImpl) messageSenderAttribution(ctx context.Context, f domain.RepoFactory, msg *domain.Message, senderAcus string) (senderType, senderID, senderName string) {
	if msg.SenderAgentConfigID != nil && *msg.SenderAgentConfigID != "" {
		name := ""
		if msg.SenderAgentName != nil {
			name = *msg.SenderAgentName
		}
		return string(constants.NotificationSenderTypeAgent), *msg.SenderAgentConfigID, name
	}
	name := ""
	if contact, apiErr := f.NewNotificationRepo().ResolveRecipientContact(ctx, senderAcus); apiErr == nil {
		name = contact.Name
	}
	return string(constants.NotificationSenderTypeUser), senderAcus, name
}

// effectivePreference resolves a recipient's channel preference for a category, defaulting to in-app on; email/push off; digest off when no preference row exists.
func (s *conversationSvcImpl) effectivePreference(ctx context.Context, prefRepo domain.NotificationPreferenceRepo, accountUserID, category string) domain.EffectiveNotificationPreference {
	eff, apiErr := prefRepo.GetEffective(ctx, accountUserID, category)
	if apiErr != nil || eff == nil {
		return domain.DefaultEffectiveNotificationPreference()
	}
	return *eff
}

// enqueueChatEmail writes a chat-notification email command to the (transaction-bound) outbox for a recipient whose email channel is enabled. A recipient without a resolvable email is skipped.
func (s *conversationSvcImpl) enqueueChatEmail(ctx context.Context, f domain.RepoFactory, outbox messaging.OutboxRepo, recipientAccountUserID, accountID, senderName, conversationTitle string, preview *string) *apierror.APIError {
	contact, apiErr := f.NewNotificationRepo().ResolveRecipientContact(ctx, recipientAccountUserID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil // no recipient row — nothing to email
		}
		return apiErr
	}
	if contact.Email == "" {
		return nil
	}
	displaySender := senderName
	if displaySender == "" {
		displaySender = "Someone"
	}
	previewText := ""
	if preview != nil {
		previewText = *preview
	}
	acctID := accountID
	payload := messaging.EmailSendData{
		To:         []string{contact.Email},
		Subject:    "New message from " + displaySender,
		TemplateID: constants.EmailTemplateChatMessage,
		Params: map[string]any{
			"SenderName":        displaySender,
			"ConversationTitle": conversationTitle,
			"MessagePreview":    previewText,
		},
		AccountID: &acctID,
	}
	return enqueueEmailVia(ctx, outbox, payload)
}

// dispatchAgents reacts to a human message in a conversation that has agent participants: when a participant's trigger policy fires, an agent run should be started and its reply posted back into this conversation as that agent participant. Starting the run (and wiring the reply via message.agent_run_id) is implemented in a later step; for now this only evaluates the triggers.
// Loop guard: it never fires on a message authored by an agent (no agent↔agent loops).
func (s *conversationSvcImpl) dispatchAgents(ctx context.Context, f domain.RepoFactory, conversationID, accountID, senderParticipantType, senderAcus string, msg *domain.Message) *apierror.APIError {
	if senderParticipantType == string(constants.ParticipantTypeAgent) {
		return nil
	}
	participants, apiErr := f.NewParticipantRepo().List(ctx, conversationID)
	if apiErr != nil {
		return apiErr
	}
	body := ""
	if msg.Body != nil {
		body = *msg.Body
	}
	// The message handed to the agent as its trigger. For an inbound email the sender is an external
	// customer (not a participant), so attribute the body with who wrote it — otherwise the agent replies
	// blind, not knowing the customer. Kept separate from `body` so trigger-keyword matching still runs
	// against the raw customer text, not the prefix.
	agentMessage := body
	if name, addr := externalSenderFromMetadata(msg.Metadata); name != "" || addr != "" {
		agentMessage = fmt.Sprintf("%s (received via email) wrote:\n\n%s", externalSenderLabel(name, addr), body)
	}
	// Replying directly to an agent's message addresses that agent — continue the run that produced the replied-to message (no re-mention needed), iMessage-style. Resolve the reply target once.
	replyAgentConfigID, replyRunID := s.replyTargetAgent(ctx, f, conversationID, accountID, msg)

	outbox := f.NewOutboxRepo()
	// Recent thread context, built once on the first firing agent (most messages trigger nothing).
	var history []chatHistoryEntry
	historyBuilt := false
	for _, p := range participants {
		if p.ParticipantType != string(constants.ParticipantTypeAgent) || p.Membership != string(constants.ParticipantMembershipActive) {
			continue
		}
		if p.AgentConfigID == nil || *p.AgentConfigID == "" {
			continue
		}
		policy := constants.AgentTriggerPolicyMention
		if p.AgentTriggerPolicy != nil && *p.AgentTriggerPolicy != "" {
			policy = constants.AgentTriggerPolicy(*p.AgentTriggerPolicy)
		}
		repliedToThisAgent := replyAgentConfigID != "" && replyAgentConfigID == *p.AgentConfigID
		if !repliedToThisAgent && !agentTriggerFires(policy, p.AgentTriggerKeywords, body) {
			continue
		}
		// A reply continues the replied-to message's run; a mention/keyword starts a fresh run.
		continueRunID := ""
		if repliedToThisAgent && replyRunID != "" {
			continueRunID = replyRunID
		}
		// Seed recent thread context (built once across firing agents). When the run actually resumes it ignores this — it carries its own transcript — but agent-service falls through to a fresh run whenever the replied-to run can't be resumed: terminal (failed / cancelled / completed) or diverged by private agent-run-console turns the conversation never saw. That fresh run needs the conversation history so it picks up the whole thread, exactly as a brand-new mention would, rather than starting from only the reply message.
		if !historyBuilt {
			history = s.buildChatHistory(ctx, conversationID, accountID, msg.Sequence)
			historyBuilt = true
		}
		hist := chatHistoryForAgent(history, *p.AgentConfigID)
		// The agent's reply threads under the message that triggered this turn: a continuation threads under the user's reply (keeping the thread growing), and a fresh directed trigger (mention or keyword) threads under that message. An "always" agent answers every message, so threading each fresh reply would be noise — it replies inline instead.
		triggerMessageID := ""
		if continueRunID != "" ||
			policy == constants.AgentTriggerPolicyMention ||
			policy == constants.AgentTriggerPolicyKeyword {
			triggerMessageID = msg.ID
		}
		// Start (or continue) a chat-linked agent run; agent-service executes it and posts the reply back via NotificationCmdAgentReply.
		if apiErr := enqueueChatRunVia(ctx, outbox, messaging.AgentChatRunData{
			AccountID:         accountID,
			AgentDefinitionID: *p.AgentConfigID,
			ConversationID:    conversationID,
			TriggerMessageID:  triggerMessageID,
			Message:           agentMessage,
			History:           hist,
			ContinueRunID:     continueRunID,
		}); apiErr != nil {
			return apiErr
		}
		// Show "<agent> is typing" to conversation subscribers while the run produces its reply.
		s.emitAgentTyping(ctx, conversationID, accountID, *p.AgentConfigID)
	}
	_ = senderAcus
	return nil
}

// replyTargetAgent inspects a message's reply target: if it replies to a message authored by an agent, it returns that agent's config id and the run that produced the replied-to message, so a direct reply continues that run. Returns empty strings when the message isn't a reply to an agent.
func (s *conversationSvcImpl) replyTargetAgent(ctx context.Context, f domain.RepoFactory, conversationID, accountID string, msg *domain.Message) (agentConfigID, runID string) {
	if msg.ReplyToMessageID == nil || *msg.ReplyToMessageID == "" {
		return "", ""
	}
	replied, apiErr := f.NewMessageRepo().GetByID(ctx, *msg.ReplyToMessageID, accountID)
	if apiErr != nil || replied == nil {
		return "", ""
	}
	s.resolveSenders(ctx, conversationID, accountID, []*domain.Message{replied}, false)
	if replied.SenderAgentConfigID == nil || *replied.SenderAgentConfigID == "" {
		return "", ""
	}
	rid := ""
	if replied.AgentRunID != nil {
		rid = *replied.AgentRunID
	}
	return *replied.SenderAgentConfigID, rid
}

// chatHistoryDepth bounds how many recent messages of thread context a chat-triggered agent run receives. The runner truncates/compacts further if the window is large.
const chatHistoryDepth = 20

// chatHistoryEntry is a per-conversation (agent-agnostic) history turn; the agent-relative role is applied per-agent in chatHistoryForAgent.
type chatHistoryEntry struct {
	agentConfigID string // authoring agent's config id; "" when a person authored it
	name          string // the person's display name when resolvable
	body          string
}

// buildChatHistory loads the recent conversational turns preceding beforeSeq (oldest-first), so a chat-triggered agent can follow the thread. System events, deletes, and empty/link-only messages are skipped; person senders are name-resolved (agents are name-resolved downstream at the gateway, so an agent turn carries its config id and an empty name here).
func (s *conversationSvcImpl) buildChatHistory(ctx context.Context, conversationID, accountID string, beforeSeq int64) []chatHistoryEntry {
	before := beforeSeq
	rows, apiErr := s.repoFactory.NewMessageRepo().List(ctx, domain.MessageListFilter{
		ConversationID: conversationID,
		Limit:          chatHistoryDepth,
		BeforeSequence: &before,
		// An agent participant is internal (vendor-side), so it must receive the full thread — internal
		// team notes included — not the customer-visible subset. Without this the filter defaults to the
		// customer view, which strips internal messages: on an all-internal thread the agent gets an empty
		// history and cannot follow the conversation (e.g. "summarize this thread" has nothing to work with).
		IncludeInternal: true,
	})
	if apiErr != nil || len(rows) == 0 {
		return nil
	}
	s.resolveSenders(ctx, conversationID, accountID, rows, false)
	nameCache := make(map[string]string)
	resolveName := func(acus string) string {
		if acus == "" {
			return ""
		}
		if n, ok := nameCache[acus]; ok {
			return n
		}
		n := ""
		if c, cErr := s.repoFactory.NewNotificationRepo().ResolveRecipientContact(ctx, acus); cErr == nil {
			n = c.Name
		}
		nameCache[acus] = n
		return n
	}
	// List returns newest-first; walk in reverse for chronological order.
	entries := make([]chatHistoryEntry, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		m := rows[i]
		if m.DeletedAt != nil || m.Kind == string(constants.MessageKindSystemEvent) {
			continue
		}
		text := ""
		if m.Body != nil {
			text = strings.TrimSpace(*m.Body)
		}
		if text == "" {
			// A link- or attachment-only message carries no body; synthesize a short stand-in so the agent still sees that something was shared (it would otherwise be invisible in the thread).
			text = describeBodylessMessage(m)
		}
		if text == "" {
			continue
		}
		e := chatHistoryEntry{body: text}
		if m.SenderAgentConfigID != nil && *m.SenderAgentConfigID != "" {
			e.agentConfigID = *m.SenderAgentConfigID
		} else if m.SenderAccountUserID != nil {
			e.name = resolveName(*m.SenderAccountUserID)
		} else if m.SenderDisplayName != nil && *m.SenderDisplayName != "" {
			// An inbound email turn: attribute it to the external sender resolveSenders surfaced, so the agent knows who wrote each prior message rather than seeing an unattributed body.
			e.name = *m.SenderDisplayName
		}
		entries = append(entries, e)
	}
	return entries
}

// describeBodylessMessage returns a short stand-in for a message that has no text body (a shared resource link or an attachment), so chat history conveys that the share happened. Returns "" when there's nothing to describe.
func describeBodylessMessage(m *domain.Message) string {
	if m.LinkResourceType != nil && *m.LinkResourceType != "" {
		return fmt.Sprintf("[shared a %s link]", strings.ReplaceAll(*m.LinkResourceType, "_", " "))
	}
	// Preview is persisted at send time and already encodes a label for attachment-only messages (e.g. "📎 Attachment") and link-only messages ("🔗 Link").
	if m.Preview != nil {
		if p := strings.TrimSpace(*m.Preview); p != "" {
			return "[" + p + "]"
		}
	}
	return ""
}

// chatHistoryForAgent stamps the agent-relative role onto each turn: the dispatched agent's own past replies become "assistant"; everything else (people and other agents) is "user". A *different* agent's turn carries its config id (Name is left empty) so agent-service — which owns agent definitions — can resolve its real display name when it builds the run.
func chatHistoryForAgent(entries []chatHistoryEntry, agentConfigID string) []messaging.ChatHistoryMessage {
	if len(entries) == 0 {
		return nil
	}
	out := make([]messaging.ChatHistoryMessage, 0, len(entries))
	for _, e := range entries {
		if e.agentConfigID != "" && e.agentConfigID == agentConfigID {
			out = append(out, messaging.ChatHistoryMessage{Role: "assistant", Body: e.body})
			continue
		}
		out = append(out, messaging.ChatHistoryMessage{Role: "user", Name: e.name, AgentConfigID: e.agentConfigID, Body: e.body})
	}
	return out
}

// enqueueChatRunVia writes an agent chat-run command to the given (transaction-bound) outbox.
func enqueueChatRunVia(ctx context.Context, outbox messaging.OutboxRepo, data messaging.AgentChatRunData) *apierror.APIError {
	payload, err := json.Marshal(data)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal agent chat-run payload.")
	}
	msg := contracts.AmqpMessage{Data: payload}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}
	if _, err := outbox.Create(ctx, messaging.OutboxMessageInput{
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.AgentCmdChatRun),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.AgentCmdChatRun),
		Payload:     msg,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to enqueue agent chat run.")
	}
	return nil
}

// PostAgentReply posts an agent run's reply into a conversation as the agent participant. The agent participant is resolved from (ConversationID, AgentConfigID); the message is kind=agent with sender_participant_id = that participant and agent_run_id = the run (so the reply renders as the agent and can expand to its run). Service-internal (no request identity) and idempotent on the client message id. A removed/inactive agent participant silently drops the reply.
// Streaming agent reply states (persisted in message.streaming_state and echoed to clients).
const (
	messageStreamingStateStreaming = "streaming"
	messageStreamingStateComplete  = "complete"
)

func (s *conversationSvcImpl) PostAgentReply(ctx context.Context, in domain.AgentReplyInput) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.post_agent_reply")
	defer span.End()

	if in.AccountID == "" || in.ConversationID == "" || in.AgentConfigID == "" || strings.TrimSpace(in.Body) == "" {
		return nil // nothing actionable (legacy single-shot drops an empty reply)
	}
	// Resolve (and gate on) the agent participant before opening the tx, so a dropped reply never starts one. nil = the agent is no longer an active participant → drop silently.
	part, apiErr := s.resolveAgentReplyParticipant(ctx, in.ConversationID, in.AgentConfigID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if part == nil {
		return nil
	}
	if apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		msg, apiErr := s.createAgentReplyTx(txCtx, f, in, part, false)
		if apiErr != nil {
			return apiErr
		}
		if msg == nil {
			return nil // idempotent redelivery
		}
		// Bell feed for human participants (the agent is the sender; senderAcus="" excludes no human).
		return s.fanoutMessageNotifications(txCtx, f, msg, in.AccountID, "")
	}); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	// Committed — wake the enqueuer so the agent reply's realtime/bell fanout is published immediately rather than after an idle poll backoff (customer-portal streaming latency).
	s.kickOutbox()
	return nil
}

func (s *conversationSvcImpl) PostAgentReplyStart(ctx context.Context, in domain.AgentReplyInput) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.post_agent_reply_start")
	defer span.End()

	if in.AccountID == "" || in.ConversationID == "" || in.AgentConfigID == "" {
		return nil
	}
	part, apiErr := s.resolveAgentReplyParticipant(ctx, in.ConversationID, in.AgentConfigID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if part == nil {
		return nil
	}
	if apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		// streaming=true: empty body, streaming_state=streaming, no bell (deferred to Complete).
		_, apiErr := s.createAgentReplyTx(txCtx, f, in, part, true)
		return apiErr
	}); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	// Committed — wake the enqueuer so the empty streaming bubble (message.created) appears immediately; the fast direct-publish token patches then have a bubble to fill.
	s.kickOutbox()
	return nil
}

func (s *conversationSvcImpl) PostAgentReplyComplete(ctx context.Context, in domain.AgentReplyInput) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.post_agent_reply_complete")
	defer span.End()

	if in.AccountID == "" || in.ConversationID == "" || in.AgentConfigID == "" {
		return nil
	}
	// No streaming row to finalize (start never ran) → legacy single-shot create-and-complete.
	if in.MessageID == "" {
		return s.PostAgentReply(ctx, in)
	}
	// Resolve the participant up front (used only by the create-if-the-start-was-lost fallback). An existing streaming row is still finalized even if the agent has since left, so a nil participant isn't an early drop here.
	part, apiErr := s.resolveAgentReplyParticipant(ctx, in.ConversationID, in.AgentConfigID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	body := strings.TrimSpace(in.Body)
	preview := strPtrIfNotEmpty(messagePreview(&body, 0, false))
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		applied, apiErr := f.NewMessageRepo().SetStreamingBody(txCtx, in.MessageID, in.AccountID, strPtrIfNotEmpty(body), preview, messageStreamingStateComplete)
		if apiErr != nil {
			return apiErr
		}
		if !applied {
			// The row was not streaming: either already finalized (redelivery → idempotent no-op) or the start was lost and must be created now (only if the agent is still a participant).
			if existing, getErr := f.NewMessageRepo().GetByID(txCtx, in.MessageID, in.AccountID); getErr == nil && existing != nil {
				return nil
			}
			if part == nil {
				return nil
			}
			msg, createErr := s.createAgentReplyTx(txCtx, f, in, part, false)
			if createErr != nil {
				return createErr
			}
			if msg == nil {
				return nil
			}
			return s.fanoutMessageNotifications(txCtx, f, msg, in.AccountID, "")
		}
		// Finalized the in-flight row: load it for bell attribution, push the final body, fire the bell.
		msg, getErr := f.NewMessageRepo().GetByID(txCtx, in.MessageID, in.AccountID)
		if getErr != nil {
			return getErr
		}
		// A failed run resolves its streaming bubble to the error text; record the failure marker + code so the client can flag the message and react (e.g. prompt to raise the spending cap).
		if in.Failed {
			if metaErr := f.NewMessageRepo().SetMessageMetadata(txCtx, in.MessageID, in.AccountID, agentFailureMetadata(in.ErrorCode)); metaErr != nil {
				return metaErr
			}
		}
		// Resolved (not persisted) so the bell notification attributes to the agent.
		msg.SenderAgentConfigID = strPtrIfNotEmpty(in.AgentConfigID)
		msg.SenderAgentName = strPtrIfNotEmpty(in.AgentName)
		if rtErr := s.enqueueMessageUpdated(txCtx, f.NewOutboxRepo(), in.ConversationID, in.AccountID, in.MessageID, body, messageStreamingStateComplete); rtErr != nil {
			return rtErr
		}
		return s.fanoutMessageNotifications(txCtx, f, msg, in.AccountID, "")
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	// Committed — wake the enqueuer so the finalized reply (message.updated + bell) is delivered immediately rather than after an idle poll backoff.
	s.kickOutbox()
	return nil
}

func (s *conversationSvcImpl) PostAgentReplyPatch(ctx context.Context, in domain.AgentReplyPatchInput) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.post_agent_reply_patch")
	defer span.End()

	if in.AccountID == "" || in.ConversationID == "" || in.MessageID == "" {
		return nil
	}
	body := in.Body
	preview := strPtrIfNotEmpty(messagePreview(&body, 0, false))
	// Best-effort, no tx: a single guarded UPDATE. applied is false when the row is no longer streaming (already complete, or the start hasn't landed yet) — skip the push so we never regress the bubble.
	applied, apiErr := s.repoFactory.NewMessageRepo().SetStreamingBody(ctx, in.MessageID, in.AccountID, strPtrIfNotEmpty(body), preview, messageStreamingStateStreaming)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if applied {
		s.publishMessageUpdated(ctx, in.ConversationID, in.AccountID, in.MessageID, body, messageStreamingStateStreaming)
	}
	return nil
}

// resolveAgentReplyParticipant resolves the agent participant a reply is posted as. It returns nil (no error) when the agent is no longer an active participant, so the caller drops the reply silently. Kept off the tx so a dropped reply never opens one.
func (s *conversationSvcImpl) resolveAgentReplyParticipant(ctx context.Context, conversationID, agentConfigID string) (*domain.ConversationParticipant, *apierror.APIError) {
	part, apiErr := s.repoFactory.NewParticipantRepo().GetByAgentConfigID(ctx, conversationID, agentConfigID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, nil // agent no longer a participant — drop the reply
		}
		return nil, apiErr
	}
	if part.Membership != string(constants.ParticipantMembershipActive) {
		return nil, nil
	}
	return part, nil
}

// createAgentReplyTx inserts a kind=agent reply (streaming or complete) within a tx for the already- resolved participant: it locks the conversation sequence, creates the row, advances the cursor, and pushes message.created. The bell is fired by the caller (deferred to completion for a streamed reply).
// Returns nil (no message) when the row already exists (idempotent redelivery).
// agentFailureMetadata builds the message metadata recorded on an agent reply that resolves a failed run: a flag the client uses to mark the message as a failure, plus the machine-readable error code (when present) it can react to (e.g. prompt to raise the spending limit on agent_spending_cap_reached).
func agentFailureMetadata(errorCode string) json.RawMessage {
	m := map[string]any{"agent_run_failed": true}
	if errorCode != "" {
		m["error_code"] = errorCode
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return b
}

func (s *conversationSvcImpl) createAgentReplyTx(txCtx context.Context, f domain.RepoFactory, in domain.AgentReplyInput, part *domain.ConversationParticipant, streaming bool) (*domain.Message, *apierror.APIError) {
	messageID := in.MessageID
	if messageID == "" {
		var genErr *apierror.APIError
		messageID, genErr = id.GenID(id.MessageIDPrefix, nil)
		if genErr != nil {
			return nil, genErr
		}
	}

	convRepo := f.NewConversationRepo()
	seq, lockErr := convRepo.LockSequence(txCtx, in.ConversationID, in.AccountID)
	if lockErr != nil {
		if lockErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, apierror.NewResourceNotFoundError("Conversation not found.")
		}
		return nil, lockErr
	}

	now := time.Now()
	agentConfigID := in.AgentConfigID
	state := messageStreamingStateComplete
	body := strings.TrimSpace(in.Body)
	var bodyPtr, previewPtr *string
	if streaming {
		// Born empty: the body fills in via patches, finalized by Complete.
		state = messageStreamingStateStreaming
		bodyPtr = strPtrIfNotEmpty(body)
		if bodyPtr != nil {
			previewPtr = strPtrIfNotEmpty(messagePreview(&body, 0, false))
		}
	} else {
		bodyPtr = &body
		previewPtr = strPtrIfNotEmpty(messagePreview(&body, 0, false))
	}

	msg := &domain.Message{
		ID:                  messageID,
		ConversationID:      in.ConversationID,
		AccountID:           in.AccountID,
		Sequence:            seq,
		Kind:                string(constants.MessageKindAgent),
		Channel:             new(string(constants.MessageChannelMessage)),
		SenderParticipantID: &part.ID,
		AgentRunID:          strPtrIfNotEmpty(in.AgentRunID),
		ClientMessageID:     strPtrIfNotEmpty(in.ClientMessageID),
		Body:                bodyPtr,
		Preview:             previewPtr,
		ReplyToMessageID:    strPtrIfNotEmpty(in.ReplyToMessageID),
		StreamingState:      &state,
		CreatedAt:           now,
		// Resolved (not persisted) so the bell notification attributes to the agent.
		SenderAgentConfigID: &agentConfigID,
		SenderAgentName:     strPtrIfNotEmpty(in.AgentName),
	}
	if in.Failed {
		msg.Metadata = agentFailureMetadata(in.ErrorCode)
	}
	inserted, createErr := f.NewMessageRepo().Create(txCtx, msg)
	if createErr != nil {
		return nil, createErr
	}
	if !inserted {
		return nil, nil // idempotent redelivery
	}
	if advErr := convRepo.AdvanceAfterMessage(txCtx, in.ConversationID, messageID, now); advErr != nil {
		return nil, advErr
	}
	if rtErr := s.fanoutMessageRealtime(txCtx, f, in.ConversationID, in.AccountID, messageID, seq, string(constants.MessageVisibilityInternal)); rtErr != nil {
		return nil, rtErr
	}
	return msg, nil
}

// messageUpdatedDelivery builds a server-push-only message.updated event carrying the streamed body and state. It targets the conversation topic only (the live thread); a streaming patch changes no unread count, so it skips the per-recipient user-topic fan-out that message.created does. Clients patch the message in place (no refetch) from the payload.
func messageUpdatedDelivery(conversationID, accountID, messageID, body, state string) (messaging.RealtimeDeliveryData, error) {
	payload, err := json.Marshal(map[string]string{"body": body, "streaming_state": state})
	if err != nil {
		return messaging.RealtimeDeliveryData{}, err
	}
	return messaging.RealtimeDeliveryData{
		AccountID:      accountID,
		ConversationID: conversationID,
		Event:          "message.updated",
		MessageID:      messageID,
		Payload:        payload,
	}, nil
}

// enqueueMessageUpdated durably enqueues a message.updated push (used by Complete, within its tx).
func (s *conversationSvcImpl) enqueueMessageUpdated(ctx context.Context, outbox messaging.OutboxRepo, conversationID, accountID, messageID, body, state string) *apierror.APIError {
	rt, err := messageUpdatedDelivery(conversationID, accountID, messageID, body, state)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal message update payload.")
	}
	return enqueueRealtimeVia(ctx, outbox, rt)
}

// publishMessageUpdated pushes a message.updated straight to the realtime fanout, bypassing the outbox (used by the high-frequency streaming patch path — lossy by design, like typing events).
func (s *conversationSvcImpl) publishMessageUpdated(ctx context.Context, conversationID, accountID, messageID, body, state string) {
	if s.broker == nil {
		return
	}
	rt, err := messageUpdatedDelivery(conversationID, accountID, messageID, body, state)
	if err != nil {
		return
	}
	data, err := json.Marshal(rt)
	if err != nil {
		return
	}
	msg := contracts.AmqpMessage{Data: data}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}
	_ = s.broker.PublishMessage(ctx, messaging.ApplicationExchange, string(contracts.NotificationEventDelivered), msg)
}

// agentTriggerFires reports whether an agent with the given policy should fire for a message body.
func agentTriggerFires(policy constants.AgentTriggerPolicy, keywords []string, body string) bool {
	switch policy {
	case constants.AgentTriggerPolicyAlways:
		return true
	case constants.AgentTriggerPolicyKeyword:
		return bodyContainsAny(body, keywords, "")
	case constants.AgentTriggerPolicyMention:
		return bodyContainsAny(body, keywords, "@")
	default:
		return false
	}
}

// bodyContainsAny reports whether body (case-insensitively) contains any keyword, each optionally prefixed (e.g. "@" for mentions).
func bodyContainsAny(body string, keywords []string, prefix string) bool {
	lowerBody := strings.ToLower(body)
	for _, kw := range keywords {
		if kw == "" {
			continue
		}
		if strings.Contains(lowerBody, prefix+strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

func (s *conversationSvcImpl) ListMessages(ctx context.Context, input domain.ListMessagesInput) (*domain.MessagePage, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.list_messages")
	defer span.End()

	// Participant gate (active members only) — supports internal and customer actors.
	rc, apiErr := s.resolveParticipant(ctx, input.ConversationID, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	identity, accountID := rc.identity, rc.accountID

	limit := clampLimit(input.Limit, defaultMessagePageSize, maxMessagePageSize)
	before, apiErr := decodeSeqCursor(input.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rows, apiErr := s.repoFactory.NewMessageRepo().List(ctx, domain.MessageListFilter{
		ConversationID: input.ConversationID,
		Limit:          limit + 1,
		BeforeSequence: before,
		AfterSequence:  input.AfterSequence,
		// SAFETY: a customer-relation viewer never receives internal-only messages — only staff (internal actors) get the full history.
		IncludeInternal: !identity.IsRelationActor(),
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	page := &domain.MessagePage{}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		page.HasNextPage = true
		last := rows[len(rows)-1]
		next := encodeSeqCursor(last.Sequence)
		page.NextCursor = &next
	}
	s.resolveSenders(ctx, input.ConversationID, accountID, rows, identity.IsRelationActor())
	s.resolveAttachments(ctx, rows)
	page.Messages = rows
	return page, nil
}

// resolveSenders attaches presentation data to each message: the polymorphic author (account_user id
// or agent_config id) resolved from the sender participant. For a customer-relation viewer, every
// vendor-side (staff/agent) author on the case is collapsed to the single branded "Customer Service"
// alias and its real author stripped, so the customer never receives the individual operator behind it.
func (s *conversationSvcImpl) resolveSenders(ctx context.Context, conversationID, _ string, messages []*domain.Message, viewerIsRelation bool) {
	if len(messages) == 0 {
		return
	}
	// ListAll (not List): a member who left or was removed still authored their past messages, so their participant row must remain resolvable or the sender surfaces as "Unknown".
	participants, apiErr := s.repoFactory.NewParticipantRepo().ListAll(ctx, conversationID)
	if apiErr != nil {
		return
	}
	if !viewerIsRelation {
		s.hydrateCustomerParticipantContacts(ctx, participants)
	}
	byParticipant := make(map[string]*domain.ConversationParticipant, len(participants))
	for _, p := range participants {
		byParticipant[p.ID] = p
	}

	for _, m := range messages {
		var author *domain.ConversationParticipant
		if m.SenderParticipantID != nil {
			if p, ok := byParticipant[*m.SenderParticipantID]; ok {
				author = p
				// Polymorphic author: a user participant resolves to its account_user id, an agent participant to its agent_config id (hydrated to a name at the gateway).
				if p.ParticipantType == string(constants.ParticipantTypeAgent) {
					m.SenderAgentConfigID = p.AgentConfigID
				} else {
					m.SenderAccountUserID = p.AccountUserID
				}
			}
		}
		// Read-time anonymization: a customer-relation viewer sees every vendor-side (staff/agent) author
		// as the single branded "Customer Service" party, never the individual operator. The customer's own
		// messages (authored by the customer participant) are left as-is.
		if viewerIsRelation && author != nil && author.ParticipantType != string(constants.ParticipantTypeCustomer) {
			m.SenderAccountUserID = nil
			m.SenderAgentConfigID = nil
			alias := supportAliasDisplayName
			m.SenderAlias = &alias
		}
		// Staff viewers: a customer-participant author carries a cross-account account_user id the gateway
		// cannot hydrate via its vendor-scoped batch loader — resolve the contact name here instead.
		if !viewerIsRelation && author != nil && author.ParticipantType == string(constants.ParticipantTypeCustomer) {
			name := ""
			if author.AccountUserDisplayName != nil {
				name = *author.AccountUserDisplayName
			} else if author.AccountUserID != nil {
				name = s.resolveContactDisplayName(ctx, *author.AccountUserID)
			}
			if name != "" {
				m.SenderDisplayName = &name
			}
		}
		// An inbound email has no participant author (the customer is external); its sender was stashed on
		// the message metadata at ingest. Surface it as the display name so the timeline shows who wrote it.
		if author == nil {
			if name, addr := externalSenderFromMetadata(m.Metadata); name != "" || addr != "" {
				label := externalSenderLabel(name, addr)
				m.SenderDisplayName = &label
			}
		}
	}
}

// attachmentUploadExpiry bounds how long a presigned upload URL is valid.
const attachmentUploadExpiry = 15 * time.Minute

// attachmentDownloadExpiry bounds how long a presigned download URL returned on read is valid.
const attachmentDownloadExpiry = 1 * time.Hour

// CreateAttachmentUploadURL issues a presigned PUT target for a chat attachment in a conversation the caller actively participates in.
func (s *conversationSvcImpl) CreateAttachmentUploadURL(ctx context.Context, conversationID, filename, contentType string) (*domain.AttachmentUploadTarget, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.create_attachment_upload_url")
	defer span.End()

	_, callerAcus, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if filename == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("A filename is required.", "filename"))
	}

	// Only an active participant may stage an upload for a conversation.
	part, apiErr := s.repoFactory.NewParticipantRepo().Get(ctx, conversationID, callerAcus)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Conversation not found."))
		}
		return nil, tracing.Trace(span, apiErr)
	}
	if part.Membership != string(constants.ParticipantMembershipActive) {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Conversation not found."))
	}

	attachmentID, apiErr := id.GenID(id.MessageAttachmentIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	key := stagedAttachmentS3Key(accountID, conversationID, attachmentID, filename)
	uploadURL, apiErr := s.objectStore.GetPresignedPutURL(ctx, s.chatBucket, key, contentType, attachmentUploadExpiry)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	now := time.Now()
	var contentTypePtr *string
	if contentType != "" {
		contentTypePtr = &contentType
	}
	return &domain.AttachmentUploadTarget{
		Attachment: &domain.MessageAttachment{
			ID:          attachmentID,
			Kind:        string(constants.UploadedAttachmentKindForContentType(contentType)),
			Filename:    &filename,
			ContentType: contentTypePtr,
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		UploadURL: uploadURL,
		S3Key:     key,
		ExpiresAt: now.Add(attachmentUploadExpiry),
	}, nil
}

// stagedAttachmentPrefix is the key prefix for not-yet-attached uploads. A bucket lifecycle rule expires anything under it after a short grace, so presigned-but-never-sent uploads self-reclaim.
const stagedAttachmentPrefix = "staged/"

// permanentAttachmentPrefix is the key prefix for attachments promoted on send; retained until the message is hard-purged (the reaper deletes these objects then the rows).
const permanentAttachmentPrefix = "chat/"

// stagedAttachmentS3Key builds the staging key a client uploads to, scoped to the account + conversation so a caller can only attach objects they staged for that conversation.
func stagedAttachmentS3Key(accountID, conversationID, attachmentID, filename string) string {
	return fmt.Sprintf("%s%s/%s/%s/%s", stagedAttachmentPrefix, accountID, conversationID, attachmentID, path.Base(filename))
}

// promoteStagedKey maps a validated staging key to its permanent key by swapping the prefix.
func promoteStagedKey(stagedKey string) string {
	return permanentAttachmentPrefix + strings.TrimPrefix(stagedKey, stagedAttachmentPrefix)
}

// buildAttachments validates each send-message attachment and returns the rows to persist. Uploaded kinds (file/image) are verified to exist under the caller's staging prefix, then promoted (copied)
// to a permanent key so the row references a durable object while the staged copy lifecycle-expires.
func (s *conversationSvcImpl) buildAttachments(ctx context.Context, conversationID, accountID, messageID string, inputs []domain.AttachmentInput) ([]*domain.MessageAttachment, *apierror.APIError) {
	if len(inputs) == 0 {
		return nil, nil
	}
	stagedPrefix := fmt.Sprintf("%s%s/%s/", stagedAttachmentPrefix, accountID, conversationID)
	out := make([]*domain.MessageAttachment, 0, len(inputs))
	for _, in := range inputs {
		kind := constants.MessageAttachmentKind(in.Kind)
		if !kind.IsValid() {
			return nil, apierror.NewParameterInvalidError("An invalid attachment kind was supplied.", "attachments")
		}
		attachmentID, apiErr := id.GenID(id.MessageAttachmentIDPrefix, nil)
		if apiErr != nil {
			return nil, apiErr
		}
		a := &domain.MessageAttachment{
			ID:          attachmentID,
			MessageID:   messageID,
			AccountID:   accountID,
			Kind:        in.Kind,
			Filename:    in.Filename,
			ContentType: in.ContentType,
			SizeBytes:   in.SizeBytes,
		}
		switch {
		case kind.IsUploaded():
			if in.S3Key == nil || *in.S3Key == "" {
				return nil, apierror.NewParameterMissingError("An uploaded attachment requires an s3_key.", "attachments")
			}
			if !strings.HasPrefix(*in.S3Key, stagedPrefix) {
				return nil, apierror.NewParameterInvalidError("The attachment does not belong to this conversation.", "attachments")
			}
			exists, apiErr := s.objectStore.FileExists(ctx, s.chatBucket, *in.S3Key)
			if apiErr != nil {
				return nil, apiErr
			}
			if !exists {
				return nil, apierror.NewParameterInvalidError("The uploaded attachment was not found. Upload it before sending.", "attachments")
			}
			// Promote the staged object to its permanent key; the staged copy expires via lifecycle.
			permanentKey := promoteStagedKey(*in.S3Key)
			if apiErr := s.objectStore.Copy(ctx, s.chatBucket, *in.S3Key, permanentKey); apiErr != nil {
				return nil, apiErr
			}
			a.S3Key = &permanentKey
		case kind == constants.MessageAttachmentKindLink:
			if in.URL == nil || *in.URL == "" {
				return nil, apierror.NewParameterMissingError("A link attachment requires a url.", "attachments")
			}
			a.URL = in.URL
		case kind == constants.MessageAttachmentKindResource:
			if in.ResourceType == nil || *in.ResourceType == "" || in.ResourceID == nil || *in.ResourceID == "" {
				return nil, apierror.NewParameterMissingError("A resource attachment requires a resource_type and resource_id.", "attachments")
			}
			a.ResourceType = in.ResourceType
			a.ResourceID = in.ResourceID
		}
		out = append(out, a)
	}
	return out, nil
}

// resolveAttachments batch-loads attachments for the given messages and presigns a download URL for each stored object. Best-effort: a presign failure leaves that attachment's URL unset.
func (s *conversationSvcImpl) resolveAttachments(ctx context.Context, messages []*domain.Message) {
	if len(messages) == 0 {
		return
	}
	ids := make([]string, 0, len(messages))
	byID := make(map[string]*domain.Message, len(messages))
	for _, m := range messages {
		ids = append(ids, m.ID)
		byID[m.ID] = m
		m.Attachments = nil
	}
	attachments, apiErr := s.repoFactory.NewMessageAttachmentRepo().ListByMessageIDs(ctx, ids)
	if apiErr != nil {
		return
	}
	for _, a := range attachments {
		if a.S3Key != nil && *a.S3Key != "" {
			if url, apiErr := s.objectStore.GetPresignedURL(ctx, s.chatBucket, *a.S3Key, attachmentDownloadExpiry); apiErr == nil {
				a.URL = &url
			}
		}
		if m, ok := byID[a.MessageID]; ok {
			m.Attachments = append(m.Attachments, a)
		}
	}
}

const defaultScheduledMessageListLimit int32 = 100

// ScheduleMessage queues a message for future delivery. The caller must be an active participant with a posting role; the scheduled_for time must be in the future.
func (s *conversationSvcImpl) ScheduleMessage(ctx context.Context, input domain.CreateScheduledMessageInput) (*domain.Message, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.schedule_message")
	defer span.End()

	_, callerAcus, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if input.Body == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("A message body is required.", "body"))
	}
	if !input.ScheduledFor.After(time.Now()) {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The scheduled_at time must be in the future.", "scheduled_at"))
	}

	part, apiErr := s.repoFactory.NewParticipantRepo().Get(ctx, input.ConversationID, callerAcus)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Conversation not found."))
		}
		return nil, tracing.Trace(span, apiErr)
	}
	if part.Membership != string(constants.ParticipantMembershipActive) {
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Conversation not found."))
	}
	if part.Role == string(constants.ParticipantRoleViewer) {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("You cannot post to this conversation."))
	}

	// Recovery-point idempotency: a scheduled message carries no client dedup key, so a retry would double-schedule.
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
		// Scheduled messages are always attributed to their author; there is no post-as persona.
		scheduledID, apiErr := id.GenID(id.MessageIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		body := input.Body
		scheduledFor := input.ScheduledFor
		msg := &domain.Message{
			ID:                  scheduledID,
			ConversationID:      input.ConversationID,
			AccountID:           accountID,
			Channel:             new(string(constants.MessageChannelMessage)),
			SenderParticipantID: &part.ID,
			Body:                &body,
			Preview:             strPtrIfNotEmpty(truncatePreview(input.Body)),
			ScheduledFor:        &scheduledFor,
		}
		var result *domain.Message
		apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
			msgRepo := f.NewMessageRepo()
			if cErr := msgRepo.CreateScheduled(txCtx, msg); cErr != nil {
				return cErr
			}
			loaded, lErr := msgRepo.GetByID(txCtx, scheduledID, accountID)
			if lErr != nil {
				return lErr
			}
			result = loaded
			return cacheSuccessResponse(txCtx, f, idemKey.TypeID, result)
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, cacheErrorResponse(ctx, s.repoFactory, idemKey.TypeID, apiErr))
		}
		return result, nil
	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idemKey.RecoveryPoint))
	}
}

// ListScheduledMessages returns the caller's scheduled (not-yet-sent) messages in a conversation, soonest first.
func (s *conversationSvcImpl) ListScheduledMessages(ctx context.Context, conversationID string) ([]*domain.Message, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.list_scheduled_messages")
	defer span.End()

	// Participant gate — supports internal and customer actors, mirroring ListMessages. Using caller()
	// here 403'd customer relation actors (they have no account_user in the vendor account), even though
	// the customer portal legitimately opens the thread. A customer has no scheduled messages, so the
	// per-sender query (scoped to rc.senderAcus, which is "" for customers) returns an empty list.
	rc, apiErr := s.resolveParticipant(ctx, conversationID, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewMessageRepo().ListScheduledByConversation(ctx, conversationID, rc.accountID, rc.senderAcus, defaultScheduledMessageListLimit)
}

// CancelScheduledMessage cancels a scheduled message the caller created.
func (s *conversationSvcImpl) CancelScheduledMessage(ctx context.Context, scheduledID string) (*domain.Message, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.cancel_scheduled_message")
	defer span.End()

	_, callerAcus, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	repo := s.repoFactory.NewMessageRepo()
	canceled, apiErr := repo.CancelScheduled(ctx, scheduledID, accountID, callerAcus)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !canceled {
		// Either it does not exist / isn't the caller's, or it is no longer scheduled.
		existing, getErr := repo.GetByID(ctx, scheduledID, accountID)
		if getErr != nil {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Scheduled message not found."))
		}
		return nil, tracing.Trace(span, apierror.NewValidationError("This scheduled message can no longer be canceled (status: "+existing.Status+")."))
	}
	return repo.GetByID(ctx, scheduledID, accountID)
}

// DeliverDueScheduledMessages promotes each due scheduled message into a sent timeline message in place. It runs without request identity (called by the scheduler worker under a lease). Each delivery re-checks that the conversation still exists and the sender is still an active participant; otherwise the scheduled message is canceled with a reason rather than sent.
func (s *conversationSvcImpl) DeliverDueScheduledMessages(ctx context.Context, limit int32) (int, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.deliver_due_scheduled_messages")
	defer span.End()

	due, apiErr := s.repoFactory.NewMessageRepo().ListDueScheduled(ctx, limit)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	delivered := 0
	for _, sm := range due {
		if s.deliverScheduledMessage(ctx, sm) {
			delivered++
		}
	}
	return delivered, nil
}

// deliverScheduledMessage promotes one scheduled message row to a sent timeline message in place,
// returning true when it was delivered. The promote is a compare-and-set on status='scheduled' inside
// the same transaction as the sequence assignment + fanout, so a redelivery (or a second pod across a
// lease handoff) is an idempotent no-op. Delivery failures and validation cancels are recorded on the
// row, not returned, so one bad row never blocks the batch.
func (s *conversationSvcImpl) deliverScheduledMessage(ctx context.Context, sm *domain.Message) bool {
	repo := s.repoFactory.NewMessageRepo()

	if sm.SenderParticipantID == nil || *sm.SenderParticipantID == "" {
		_ = repo.MarkScheduledFailed(ctx, sm.ID, string(constants.MessageStatusFailed), new("scheduled message has no sender participant"))
		return false
	}

	// Re-validate the conversation still exists and the sender is still an active participant.
	if _, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, sm.ConversationID, sm.AccountID); apiErr != nil {
		_ = repo.MarkScheduledFailed(ctx, sm.ID, string(constants.MessageStatusCanceled), new("conversation no longer exists"))
		return false
	}
	part, apiErr := s.repoFactory.NewParticipantRepo().GetByID(ctx, *sm.SenderParticipantID, sm.ConversationID)
	if apiErr != nil || part.Membership != string(constants.ParticipantMembershipActive) {
		_ = repo.MarkScheduledFailed(ctx, sm.ID, string(constants.MessageStatusCanceled), new("sender is no longer an active participant"))
		return false
	}

	var senderAcus string
	if part.AccountUserID != nil {
		senderAcus = *part.AccountUserID
	}

	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		convRepo := f.NewConversationRepo()
		seq, lockErr := convRepo.LockSequence(txCtx, sm.ConversationID, sm.AccountID)
		if lockErr != nil {
			return lockErr
		}
		applied, promErr := f.NewMessageRepo().PromoteScheduled(txCtx, sm.ID, seq)
		if promErr != nil {
			return promErr
		}
		if !applied {
			// A prior tick already promoted this row; idempotent no-op (don't re-advance/re-fanout).
			return nil
		}
		now := time.Now()
		sm.Sequence = seq
		sm.Status = string(constants.MessageStatusSent)
		if advErr := convRepo.AdvanceAfterMessage(txCtx, sm.ConversationID, sm.ID, now); advErr != nil {
			return advErr
		}
		if senderAcus != "" {
			if curErr := f.NewParticipantRepo().AdvanceReadCursor(txCtx, sm.ConversationID, senderAcus, sm.ID, seq); curErr != nil {
				return curErr
			}
		}
		if rtErr := s.fanoutMessageRealtime(txCtx, f, sm.ConversationID, sm.AccountID, sm.ID, seq, string(constants.MessageVisibilityInternal)); rtErr != nil {
			return rtErr
		}
		if bellErr := s.fanoutMessageNotifications(txCtx, f, sm, sm.AccountID, senderAcus); bellErr != nil {
			return bellErr
		}
		return s.dispatchAgents(txCtx, f, sm.ConversationID, sm.AccountID, part.ParticipantType, senderAcus, sm)
	})
	if apiErr != nil {
		// Release any claim and leave it scheduled to retry next tick.
		_ = repo.MarkScheduledFailed(ctx, sm.ID, string(constants.MessageStatusScheduled), new(apiErr.PublicMessage))
		return false
	}
	return true
}

func (s *conversationSvcImpl) MarkConversationRead(ctx context.Context, conversationID string, upToSequence int64) (int64, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.mark_read")
	defer span.End()

	rc, apiErr := s.resolveParticipant(ctx, conversationID, false)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	identity, part, accountID := rc.identity, rc.part, rc.accountID

	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, accountID)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	maxSeq := conv.NextSequence - 1
	target := upToSequence
	if target > maxSeq {
		target = maxSeq
	}

	if target > part.LastReadSequence {
		if apiErr := s.repoFactory.NewParticipantRepo().AdvanceReadCursorByID(ctx, part.ID, "", target); apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}
		// The cursor genuinely advanced — tell the other participants so their read receipts ("Seen")
		// update live. Reuse conversation.updated (clients refetch the conversation, picking up every
		// participant's read cursor). Gated on an actual advance so routine re-marks — e.g. focusing an
		// already-read thread — don't fan a broadcast out to everyone.
		s.fanoutConversationEvent(ctx, conversationID, accountID, "conversation.updated", "")
	}

	effectiveCursor := part.LastReadSequence
	if target > effectiveCursor {
		effectiveCursor = target
	}
	unread := unreadFrom(conv.NextSequence, effectiveCursor)

	// Reading a conversation in-thread auto-dismisses its bell notifications for this recipient, so already-seen chat alerts don't stack in the bell. Best-effort: a failure here must not fail the read.
	dismissed := int64(0)
	if part.AccountUserID != nil && *part.AccountUserID != "" {
		if n, apiErr := s.repoFactory.NewNotificationRepo().DismissByConversation(ctx, *part.AccountUserID, conversationID); apiErr != nil {
			slog.WarnContext(ctx, "mark_read: failed to dismiss conversation notifications", "error", apiErr, "conversation_id", conversationID)
		} else {
			dismissed = n
		}
	}

	// Best-effort: nudge the caller's other tabs to refresh the badge.
	_ = enqueueRealtimeVia(ctx, s.repoFactory.NewOutboxRepo(), messaging.RealtimeDeliveryData{
		AccountID:       accountID,
		RecipientUserID: identity.Actor.ID,
		ConversationID:  conversationID,
		Event:           "unread.changed",
	})
	// If any bell notifications were withdrawn, nudge the user's tabs to refresh the bell feed/count.
	// A non-message event maps to the notification WS type on the gateway.
	if dismissed > 0 {
		_ = enqueueRealtimeVia(ctx, s.repoFactory.NewOutboxRepo(), messaging.RealtimeDeliveryData{
			AccountID:       accountID,
			RecipientUserID: identity.Actor.ID,
			Event:           "notification.dismissed",
		})
	}
	return unread, nil
}

func (s *conversationSvcImpl) IsParticipant(ctx context.Context, conversationID, userID, accountID string) (bool, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.is_participant")
	defer span.End()

	notifRepo := s.repoFactory.NewNotificationRepo()
	participantRepo := s.repoFactory.NewParticipantRepo()

	accountUserID, apiErr := notifRepo.ResolveAccountUserID(ctx, userID, accountID)
	if apiErr == nil {
		part, partErr := participantRepo.Get(ctx, conversationID, accountUserID)
		if partErr != nil {
			if partErr.Code == apierror.ErrorCodeResourceNotFound {
				return false, nil
			}
			return false, tracing.Trace(span, partErr)
		}
		return part.Membership == string(constants.ParticipantMembershipActive), nil
	}
	if apiErr.Code != apierror.ErrorCodeResourceNotFound {
		return false, tracing.Trace(span, apiErr)
	}

	// Customer/supplier relation actors are not account_users of the vendor account but participate via a relation_account_id row. The WS handler validates them against the vendor account (target), so membership must be resolved the same way as resolveParticipant.
	participants, listErr := participantRepo.List(ctx, conversationID)
	if listErr != nil {
		return false, tracing.Trace(span, listErr)
	}
	for _, p := range participants {
		if p.Membership != string(constants.ParticipantMembershipActive) {
			continue
		}
		if p.ParticipantType != string(constants.ParticipantTypeCustomer) {
			continue
		}
		if p.RelationAccountID == nil || *p.RelationAccountID == "" {
			continue
		}
		if _, relErr := notifRepo.ResolveAccountUserID(ctx, userID, *p.RelationAccountID); relErr == nil {
			return true, nil
		}
	}
	return false, nil
}

// SendTyping publishes an ephemeral "typing" event for the conversation directly to the realtime fanout, bypassing the outbox. Typing is high-frequency and disposable: persisting one outbox row per keystroke would pollute the outbox and its metrics (§5, §8). A dropped typing event is harmless, so publish failures are swallowed after authz succeeds.
// emitAgentTyping publishes a best-effort "agent is typing" realtime event so conversation subscribers see the agent working between the triggering message and its reply. Mirrors SendTyping but marks the typer as an agent (carrying its agent_config_id) so clients render the agent identity and hold the indicator longer; a run outlasts the human typing TTL, and the reply landing retires it.
func (s *conversationSvcImpl) emitAgentTyping(ctx context.Context, conversationID, accountID, agentConfigID string) {
	if s.broker == nil || agentConfigID == "" {
		return
	}
	payload, _ := json.Marshal(map[string]string{"account_user_id": agentConfigID, "is_agent": "true"})
	data, err := json.Marshal(messaging.RealtimeDeliveryData{
		AccountID:      accountID,
		ConversationID: conversationID,
		Event:          "typing",
		Payload:        payload,
	})
	if err != nil {
		return
	}
	msg := contracts.AmqpMessage{Data: data}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}
	_ = s.broker.PublishMessage(ctx, messaging.ApplicationExchange, string(contracts.NotificationEventDelivered), msg)
}

func (s *conversationSvcImpl) SendTyping(ctx context.Context, conversationID string) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.send_typing")
	defer span.End()

	// resolveParticipant (not requireParticipant) so customer/supplier relation actors can broadcast typing in their support thread too; their senderAcus is empty, which clients treat as the non-staff typer.
	rc, apiErr := s.resolveParticipant(ctx, conversationID, false)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	callerAcus, accountID := rc.senderAcus, rc.accountID
	if s.broker == nil {
		return nil // realtime publishing unavailable; typing is best-effort
	}

	// Carry the typer's account_user id so clients can label "X is typing" and ignore their own echo.
	payload, _ := json.Marshal(map[string]string{"account_user_id": callerAcus})
	data, err := json.Marshal(messaging.RealtimeDeliveryData{
		AccountID:      accountID,
		ConversationID: conversationID,
		Event:          "typing",
		Payload:        payload,
	})
	if err != nil {
		return nil
	}
	msg := contracts.AmqpMessage{Data: data}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}
	if err := s.broker.PublishMessage(ctx, messaging.ApplicationExchange, string(contracts.NotificationEventDelivered), msg); err != nil {
		tracing.Trace(span, apierror.NewInternalError(err, "Failed to publish typing event."))
	}
	return nil
}

// createGroup creates a group conversation with the caller as owner and the given members.
func (s *conversationSvcImpl) createGroup(ctx context.Context, input domain.CreateConversationInput, callerAcus, accountID string) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.create_group")
	defer span.End()

	// Seed the participant set from explicitly-requested users plus, when a reusable roster is named, a snapshot of that roster's members. The roster is provenance only: its members are copied in here and the conversation is independent thereafter (later roster edits never reach it).
	seedUserIDs := append([]string(nil), input.ParticipantAccountUserIDs...)
	var seedAgentIDs []string
	if input.GroupID != nil && *input.GroupID != "" {
		group, apiErr := s.repoFactory.NewMessagingGroupRepo().Get(ctx, *input.GroupID, accountID)
		if apiErr != nil {
			if apiErr.Code == apierror.ErrorCodeResourceNotFound {
				return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The group does not exist.", "group_id"))
			}
			return nil, tracing.Trace(span, apiErr)
		}
		groupMembers, apiErr := s.repoFactory.NewMessagingGroupRepo().ListMembers(ctx, group.ID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		for _, gm := range groupMembers {
			switch gm.MemberType {
			case domain.MessagingGroupMemberTypeUser:
				if gm.AccountUserID != nil {
					seedUserIDs = append(seedUserIDs, *gm.AccountUserID)
				}
			case domain.MessagingGroupMemberTypeAgent:
				if gm.AgentConfigID != nil {
					seedAgentIDs = append(seedAgentIDs, *gm.AgentConfigID)
				}
			}
		}
	}

	members := make([]string, 0, len(seedUserIDs))
	seen := map[string]struct{}{callerAcus: {}}
	for _, m := range seedUserIDs {
		if m == "" {
			continue
		}
		if _, dup := seen[m]; dup {
			continue
		}
		seen[m] = struct{}{}
		if _, apiErr := s.repoFactory.NewNotificationRepo().ResolveUserID(ctx, m); apiErr != nil {
			if apiErr.Code == apierror.ErrorCodeResourceNotFound {
				return nil, tracing.Trace(span, apierror.NewParameterInvalidError("A participant does not exist.", "participant_account_user_ids"))
			}
			return nil, tracing.Trace(span, apiErr)
		}
		members = append(members, m)
	}
	// De-duplicate agent members (a roster cannot contain the same agent twice, but the explicit set could overlap a future param).
	agents := make([]string, 0, len(seedAgentIDs))
	seenAgents := map[string]struct{}{}
	for _, a := range seedAgentIDs {
		if a == "" {
			continue
		}
		if _, dup := seenAgents[a]; dup {
			continue
		}
		seenAgents[a] = struct{}{}
		agents = append(agents, a)
	}
	// A group normally needs at least one other participant — a human member or an agent — but a record-anchored discussion may start solo: the creator posts and loops in the team afterward (via @mention or by adding participants).
	hasTopic := input.TopicResourceID != nil && *input.TopicResourceID != ""
	if len(members) == 0 && len(agents) == 0 && !hasTopic {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("A group requires at least one other participant.", "participant_account_user_ids"))
	}

	conversationID, apiErr := id.GenID(id.ConversationIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if createErr := f.NewConversationRepo().Create(txCtx, conversationID, &input, accountID); createErr != nil {
			return createErr
		}
		partRepo := f.NewParticipantRepo()
		// Creator is the owner; everyone else joins as a member.
		roleByAcus := map[string]string{callerAcus: string(constants.ParticipantRoleOwner)}
		ordered := append([]string{callerAcus}, members...)
		for _, acus := range ordered {
			role := roleByAcus[acus]
			if role == "" {
				role = string(constants.ParticipantRoleMember)
			}
			pid, genErr := id.GenID(id.ConversationParticipantIDPrefix, nil)
			if genErr != nil {
				return genErr
			}
			acusCopy := acus
			if pErr := partRepo.Create(txCtx, &domain.ConversationParticipant{
				ID:              pid,
				ConversationID:  conversationID,
				AccountID:       accountID,
				ParticipantType: string(constants.ParticipantTypeUser),
				AccountUserID:   &acusCopy,
				Role:            role,
			}); pErr != nil {
				return pErr
			}
		}
		// Roster agent members are seated with the safe default trigger (mention); their per-conversation trigger config can be changed afterward like any other agent participant.
		for _, agentConfigID := range agents {
			pid, genErr := id.GenID(id.ConversationParticipantIDPrefix, nil)
			if genErr != nil {
				return genErr
			}
			if pErr := partRepo.CreateAgent(txCtx, pid, accountID, &domain.AddAgentParticipantInput{
				ConversationID: conversationID,
				AgentConfigID:  agentConfigID,
				TriggerPolicy:  string(constants.AgentTriggerPolicyMention),
			}); pErr != nil {
				return pErr
			}
		}
		return nil
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	// Each initial member (not the creator) gets a chat.added bell (§12.12).
	conv := &domain.Conversation{ID: conversationID, AccountID: accountID, Title: input.Title}
	for _, m := range members {
		if apiErr := s.writeAddedNotification(ctx, conv, m); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}
	return s.loadConversation(ctx, conversationID, callerAcus, accountID)
}

func (s *conversationSvcImpl) UpdateConversation(ctx context.Context, conversationID string, title *string, status *string, clearTitle bool) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.update")
	defer span.End()

	// Map the lifecycle status enum to the backing is_archived flag.
	var isArchived *bool
	if status != nil {
		st := constants.ConversationStatus(*status)
		if !st.IsValid() {
			return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The status is invalid.", "status"))
		}
		archived := st == constants.ConversationStatusArchived
		isArchived = &archived
	}

	callerAcus, accountID, part, apiErr := s.requireParticipant(ctx, conversationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if conv.Type == string(constants.ConversationTypeDM) {
		return nil, tracing.Trace(span, apierror.NewValidationError("A direct message cannot be renamed or archived."))
	}
	if !roleAllows(part.Role, constants.ParticipantRoleOwner, constants.ParticipantRoleAdmin) {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("Only an owner or admin can update this conversation."))
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		convRepo := f.NewConversationRepo()
		if apiErr := convRepo.Update(txCtx, conversationID, accountID, title, isArchived, clearTitle); apiErr != nil {
			return apiErr
		}
		updated, apiErr := convRepo.GetByID(txCtx, conversationID, accountID)
		if apiErr != nil {
			return apiErr
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeConversation,
			ResourceID:   conversationID,
			Changes:      audit.ComputeChanges(conv, updated),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	s.fanoutConversationEvent(ctx, conversationID, accountID, "conversation.updated", "")
	return s.loadConversation(ctx, conversationID, callerAcus, accountID)
}

func (s *conversationSvcImpl) AddParticipant(ctx context.Context, conversationID, targetAccountUserID, role string) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.add_participant")
	defer span.End()

	callerAcus, accountID, part, apiErr := s.requireParticipant(ctx, conversationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if conv.Type == string(constants.ConversationTypeDM) {
		return nil, tracing.Trace(span, apierror.NewValidationError("Participants cannot be added to a direct message."))
	}
	if !roleAllows(part.Role, constants.ParticipantRoleOwner, constants.ParticipantRoleAdmin) {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("Only an owner or admin can add participants."))
	}
	newRole := constants.ParticipantRole(role)
	if newRole == "" {
		newRole = constants.ParticipantRoleMember
	}
	if !newRole.IsValid() || newRole == constants.ParticipantRoleOwner {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The role is invalid.", "role"))
	}
	if targetAccountUserID == "" || targetAccountUserID == callerAcus {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The participant is invalid.", "account_user_id"))
	}
	if _, apiErr := s.repoFactory.NewNotificationRepo().ResolveUserID(ctx, targetAccountUserID); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The participant does not exist.", "account_user_id"))
		}
		return nil, tracing.Trace(span, apiErr)
	}

	partRepo := s.repoFactory.NewParticipantRepo()
	existing, getErr := partRepo.Get(ctx, conversationID, targetAccountUserID)
	if getErr != nil && getErr.Code != apierror.ErrorCodeResourceNotFound {
		return nil, tracing.Trace(span, getErr)
	}
	// Already an active member: no mutation, no audit.
	if getErr == nil && existing.Membership == string(constants.ParticipantMembershipActive) {
		return s.loadConversation(ctx, conversationID, callerAcus, accountID)
	}
	// Reactivate a previously left/removed row, or create a fresh participant.
	reactivating := getErr == nil
	var addedParticipantID string
	if reactivating {
		addedParticipantID = existing.ID
	} else {
		pid, genErr := id.GenID(id.ConversationParticipantIDPrefix, nil)
		if genErr != nil {
			return nil, tracing.Trace(span, genErr)
		}
		addedParticipantID = pid
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txPartRepo := f.NewParticipantRepo()
		if reactivating {
			if reErr := txPartRepo.Reactivate(txCtx, conversationID, targetAccountUserID, string(newRole)); reErr != nil {
				return reErr
			}
		} else {
			if cErr := txPartRepo.Create(txCtx, &domain.ConversationParticipant{
				ID:              addedParticipantID,
				ConversationID:  conversationID,
				AccountID:       accountID,
				ParticipantType: string(constants.ParticipantTypeUser),
				AccountUserID:   &targetAccountUserID,
				Role:            string(newRole),
			}); cErr != nil {
				return cErr
			}
		}
		added, apiErr := txPartRepo.GetByID(txCtx, addedParticipantID, conversationID)
		if apiErr != nil {
			return apiErr
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeConversationParticipant,
			ResourceID:   addedParticipantID,
			Changes:      audit.ComputeChanges(nil, added),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Being added to a conversation writes a bell for the added user (§12.12).
	if apiErr := s.writeAddedNotification(ctx, conv, targetAccountUserID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	s.postSystemEvent(ctx, conversationID, accountID, addedParticipantID, "participant.added", "was added to the conversation")
	s.fanoutConversationEvent(ctx, conversationID, accountID, "conversation.updated", "")
	return s.loadConversation(ctx, conversationID, callerAcus, accountID)
}

// writeAddedNotification records a chat.added bell for a user added to a conversation.
func (s *conversationSvcImpl) writeAddedNotification(ctx context.Context, conv *domain.Conversation, addedAccountUserID string) *apierror.APIError {
	notifID, apiErr := id.GenID(id.NotificationIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}
	title := "You were added to a conversation"
	var body *string
	if conv.Title != nil && *conv.Title != "" {
		body = conv.Title
	}
	return s.repoFactory.NewNotificationRepo().Create(ctx, &domain.Notification{
		ID:                     notifID,
		AccountID:              conv.AccountID,
		RecipientAccountUserID: addedAccountUserID,
		Category:               string(constants.NotificationCategoryChatAdded),
		ConversationID:         &conv.ID,
		Title:                  title,
		Body:                   body,
		Priority:               string(constants.NotificationPriorityNormal),
	})
}

// postSystemEvent appends a system_event message (e.g. "left the conversation") authored by the affected participant, advancing the conversation sequence and fanning out a realtime message event so open threads render it as a timeline divider. Best-effort: a failure must not fail the membership change that triggered it, so the error is swallowed. No bell notification or agent dispatch — system events are timeline-only.
func (s *conversationSvcImpl) postSystemEvent(ctx context.Context, conversationID, accountID, senderParticipantID, eventType, body string) {
	messageID, apiErr := id.GenID(id.MessageIDPrefix, nil)
	if apiErr != nil {
		return
	}
	_ = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		convRepo := f.NewConversationRepo()
		seq, lockErr := convRepo.LockSequence(txCtx, conversationID, accountID)
		if lockErr != nil {
			return lockErr
		}
		now := time.Now()
		et := eventType
		msg := &domain.Message{
			ID:                  messageID,
			ConversationID:      conversationID,
			AccountID:           accountID,
			Sequence:            seq,
			Kind:                string(constants.MessageKindSystemEvent),
			Channel:             new(string(constants.MessageChannelMessage)),
			SenderParticipantID: &senderParticipantID,
			EventType:           &et,
			Body:                &body,
			Preview:             strPtrIfNotEmpty(truncatePreview(body)),
			CreatedAt:           now,
		}
		if _, createErr := f.NewMessageRepo().Create(txCtx, msg); createErr != nil {
			return createErr
		}
		if advErr := convRepo.AdvanceAfterMessage(txCtx, conversationID, messageID, now); advErr != nil {
			return advErr
		}
		return s.fanoutMessageRealtime(txCtx, f, conversationID, accountID, messageID, seq, string(constants.MessageVisibilityInternal))
	})
}

// PostConversationSystemEvent appends a senderless system_event message whose Body is the full sentence (e.g. "Dane approved update_customer"), rendered as a timeline divider. Timeline-only (no bell), idempotent on the client message id so a redelivered command doesn't double-post.
func (s *conversationSvcImpl) PostConversationSystemEvent(ctx context.Context, in domain.SystemEventInput) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.post_system_event")
	defer span.End()

	if in.AccountID == "" || in.ConversationID == "" || in.Body == "" {
		return nil // nothing actionable
	}

	messageID, apiErr := id.GenID(id.MessageIDPrefix, nil)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	return s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		convRepo := f.NewConversationRepo()
		seq, lockErr := convRepo.LockSequence(txCtx, in.ConversationID, in.AccountID)
		if lockErr != nil {
			if lockErr.Code == apierror.ErrorCodeResourceNotFound {
				return nil // conversation gone — drop the notice
			}
			return lockErr
		}
		now := time.Now()
		et := in.EventType
		body := in.Body
		clientMsgID := in.ClientMessageID
		msg := &domain.Message{
			ID:              messageID,
			ConversationID:  in.ConversationID,
			AccountID:       in.AccountID,
			Sequence:        seq,
			Kind:            string(constants.MessageKindSystemEvent),
			Channel:         new(string(constants.MessageChannelMessage)),
			EventType:       &et,
			Body:            &body,
			Preview:         strPtrIfNotEmpty(truncatePreview(body)),
			ClientMessageID: strPtrIfNotEmpty(clientMsgID),
			CreatedAt:       now,
		}
		inserted, createErr := f.NewMessageRepo().Create(txCtx, msg)
		if createErr != nil {
			return createErr
		}
		if !inserted {
			return nil // idempotent redelivery
		}
		if advErr := convRepo.AdvanceAfterMessage(txCtx, in.ConversationID, messageID, now); advErr != nil {
			return advErr
		}
		return s.fanoutMessageRealtime(txCtx, f, in.ConversationID, in.AccountID, messageID, seq, string(constants.MessageVisibilityInternal))
	})
}

func (s *conversationSvcImpl) RemoveParticipant(ctx context.Context, conversationID, participantID string) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.remove_participant")
	defer span.End()

	_, accountID, part, apiErr := s.requireParticipant(ctx, conversationID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, accountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if conv.Type == string(constants.ConversationTypeDM) {
		return tracing.Trace(span, apierror.NewValidationError("Participants cannot be removed from a direct message."))
	}
	if !roleAllows(part.Role, constants.ParticipantRoleOwner, constants.ParticipantRoleAdmin) {
		return tracing.Trace(span, apierror.NewAuthorizationError("Only an owner or admin can remove participants."))
	}
	target, apiErr := s.repoFactory.NewParticipantRepo().GetByID(ctx, participantID, conversationID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return tracing.Trace(span, apierror.NewResourceNotFoundError("Participant not found."))
		}
		return tracing.Trace(span, apiErr)
	}
	if target.ID == part.ID {
		return tracing.Trace(span, apierror.NewValidationError("Use leave to remove yourself."))
	}
	if target.AccountUserID == nil {
		return tracing.Trace(span, apierror.NewValidationError("This participant cannot be removed here."))
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if apiErr := f.NewParticipantRepo().SetState(txCtx, conversationID, *target.AccountUserID, string(constants.ParticipantMembershipRemoved)); apiErr != nil {
			return apiErr
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeConversationParticipant,
			ResourceID:   participantID,
			Changes:      audit.ComputeChanges(target, (*domain.ConversationParticipant)(nil)),
		})
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	s.postSystemEvent(ctx, conversationID, accountID, target.ID, "participant.removed", "was removed from the conversation")
	s.fanoutConversationEvent(ctx, conversationID, accountID, "conversation.updated", "")
	// The fanout above lists only active participants, so the just-removed user is skipped. Nudge them directly on their personal topic so their client drops the now-inaccessible thread and returns to the messages list (and unsubscribes, so the group's typing/message events stop reaching them).
	if userID, apiErr := s.repoFactory.NewNotificationRepo().ResolveUserID(ctx, *target.AccountUserID); apiErr == nil {
		_ = enqueueRealtimeVia(ctx, s.repoFactory.NewOutboxRepo(), messaging.RealtimeDeliveryData{
			AccountID:       accountID,
			RecipientUserID: userID,
			ConversationID:  conversationID,
			Event:           "conversation.updated",
		})
	}
	return nil
}

func (s *conversationSvcImpl) AddAgentParticipant(ctx context.Context, input domain.AddAgentParticipantInput) (*domain.ConversationParticipant, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.add_agent_participant")
	defer span.End()

	_, accountID, part, apiErr := s.requireParticipant(ctx, input.ConversationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, input.ConversationID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !canManageAgentsInConversation(conv, part) {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("Only an owner or admin can add an agent."))
	}
	if input.AgentConfigID == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("An agent is required.", "agent_config_id"))
	}
	if input.TriggerPolicy == "" {
		input.TriggerPolicy = string(constants.AgentTriggerPolicyMention) // safest default
	}
	if !constants.AgentTriggerPolicy(input.TriggerPolicy).IsValid() {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("An invalid trigger policy was supplied.", "trigger_policy"))
	}

	partRepo := s.repoFactory.NewParticipantRepo()
	existing, getErr := partRepo.GetByAgentConfigID(ctx, input.ConversationID, input.AgentConfigID)
	if getErr != nil && getErr.Code != apierror.ErrorCodeResourceNotFound {
		return nil, tracing.Trace(span, getErr)
	}
	// Already a participant (active or previously removed): update its trigger policy and ensure it's active. This makes add an idempotent upsert, so it doubles as "change when this agent responds" without removing and re-adding it.
	reactivating := getErr == nil
	var participantID string
	if reactivating {
		participantID = existing.ID
	} else {
		pid, genErr := id.GenID(id.ConversationParticipantIDPrefix, nil)
		if genErr != nil {
			return nil, tracing.Trace(span, genErr)
		}
		participantID = pid
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txPartRepo := f.NewParticipantRepo()
		if reactivating {
			if reErr := txPartRepo.ReactivateAgent(txCtx, participantID, &input); reErr != nil {
				return reErr
			}
		} else {
			if cErr := txPartRepo.CreateAgent(txCtx, participantID, accountID, &input); cErr != nil {
				return cErr
			}
		}
		added, apiErr := txPartRepo.GetByID(txCtx, participantID, input.ConversationID)
		if apiErr != nil {
			return apiErr
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeConversationParticipant,
			ResourceID:   participantID,
			Changes:      audit.ComputeChanges(nil, added),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	s.fanoutConversationEvent(ctx, input.ConversationID, accountID, "conversation.updated", "")
	return partRepo.GetByAgentConfigID(ctx, input.ConversationID, input.AgentConfigID)
}

func (s *conversationSvcImpl) RemoveAgentParticipant(ctx context.Context, conversationID, participantID string) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.remove_agent_participant")
	defer span.End()

	_, accountID, part, apiErr := s.requireParticipant(ctx, conversationID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, accountID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if !canManageAgentsInConversation(conv, part) {
		return tracing.Trace(span, apierror.NewAuthorizationError("Only an owner or admin can remove an agent."))
	}
	target, apiErr := s.repoFactory.NewParticipantRepo().GetByID(ctx, participantID, conversationID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return tracing.Trace(span, apierror.NewResourceNotFoundError("Agent participant not found."))
		}
		return tracing.Trace(span, apiErr)
	}
	if target.ParticipantType != string(constants.ParticipantTypeAgent) {
		return tracing.Trace(span, apierror.NewValidationError("This participant is not an agent."))
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if apiErr := f.NewParticipantRepo().SetStateByID(txCtx, participantID, conversationID, string(constants.ParticipantMembershipRemoved)); apiErr != nil {
			return apiErr
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeConversationParticipant,
			ResourceID:   participantID,
			Changes:      audit.ComputeChanges(target, (*domain.ConversationParticipant)(nil)),
		})
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	s.fanoutConversationEvent(ctx, conversationID, accountID, "conversation.updated", "")
	return nil
}

func (s *conversationSvcImpl) UpdateParticipantRole(ctx context.Context, conversationID, participantID, role string) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.update_participant_role")
	defer span.End()

	callerAcus, accountID, part, apiErr := s.requireParticipant(ctx, conversationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if part.Role != string(constants.ParticipantRoleOwner) {
		return nil, tracing.Trace(span, apierror.NewAuthorizationError("Only the owner can change roles."))
	}
	newRole := constants.ParticipantRole(role)
	if !newRole.IsValid() {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The role is invalid.", "role"))
	}
	target, apiErr := s.repoFactory.NewParticipantRepo().GetByID(ctx, participantID, conversationID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Participant not found."))
		}
		return nil, tracing.Trace(span, apiErr)
	}
	if target.AccountUserID == nil {
		return nil, tracing.Trace(span, apierror.NewValidationError("This participant's role cannot be changed here."))
	}
	roleChanged := target.Role != string(newRole)
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		txPartRepo := f.NewParticipantRepo()
		if apiErr := txPartRepo.SetRole(txCtx, conversationID, *target.AccountUserID, string(newRole)); apiErr != nil {
			return apiErr
		}
		updated, apiErr := txPartRepo.GetByID(txCtx, participantID, conversationID)
		if apiErr != nil {
			return apiErr
		}
		// A same-value role set produces an empty diff, which Publish drops as a no-op.
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeConversationParticipant,
			ResourceID:   participantID,
			Changes:      audit.ComputeChanges(target, updated),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if roleChanged {
		s.postSystemEvent(ctx, conversationID, accountID, target.ID, "participant.role_changed", roleChangeBody(newRole))
	}
	return s.loadConversation(ctx, conversationID, callerAcus, accountID)
}

// roleChangeBody is the system-event predicate for a role change (subject is the affected member).
func roleChangeBody(role constants.ParticipantRole) string {
	switch role {
	case constants.ParticipantRoleOwner:
		return "was made an owner"
	case constants.ParticipantRoleAdmin:
		return "was made an admin"
	case constants.ParticipantRoleViewer:
		return "was made a viewer"
	default:
		return "was made a member"
	}
}

func (s *conversationSvcImpl) Leave(ctx context.Context, conversationID string) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.leave")
	defer span.End()

	callerAcus, accountID, part, apiErr := s.requireParticipant(ctx, conversationID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if part.Role == string(constants.ParticipantRoleOwner) {
		return tracing.Trace(span, apierror.NewAuthorizationError("The owner cannot leave a conversation. Transfer ownership first."))
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if apiErr := f.NewParticipantRepo().Leave(txCtx, conversationID, callerAcus); apiErr != nil {
			return apiErr
		}
		// Leaving persists the caller's participant removal — audit it like RemoveParticipant.
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeConversationParticipant,
			ResourceID:   part.ID,
			Changes:      audit.ComputeChanges(part, (*domain.ConversationParticipant)(nil)),
		})
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	s.postSystemEvent(ctx, conversationID, accountID, part.ID, "participant.left", "left the conversation")
	s.fanoutConversationEvent(ctx, conversationID, accountID, "conversation.updated", "")
	return nil
}

func (s *conversationSvcImpl) Hide(ctx context.Context, conversationID string) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.hide")
	defer span.End()

	callerAcus, _, part, apiErr := s.requireParticipant(ctx, conversationID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if part.Role == string(constants.ParticipantRoleOwner) {
		return tracing.Trace(span, apierror.NewAuthorizationError("The owner cannot hide a conversation."))
	}
	if apiErr := s.repoFactory.NewParticipantRepo().Hide(ctx, conversationID, callerAcus); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (s *conversationSvcImpl) Unhide(ctx context.Context, conversationID string) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.unhide")
	defer span.End()

	callerAcus, _, _, apiErr := s.requireParticipant(ctx, conversationID)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	if apiErr := s.repoFactory.NewParticipantRepo().Unhide(ctx, conversationID, callerAcus); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (s *conversationSvcImpl) SetMute(ctx context.Context, conversationID string, muted bool, mutedUntil *time.Time) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.set_mute")
	defer span.End()

	callerAcus, accountID, _, apiErr := s.requireParticipant(ctx, conversationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := s.repoFactory.NewParticipantRepo().SetMute(ctx, conversationID, callerAcus, muted, mutedUntil); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.loadConversation(ctx, conversationID, callerAcus, accountID)
}

func (s *conversationSvcImpl) Block(ctx context.Context, blockedAccountUserID string) (*domain.MessageBlock, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.block")
	defer span.End()

	_, callerAcus, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if blockedAccountUserID == "" || blockedAccountUserID == callerAcus {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The user to block is invalid.", "account_user_id"))
	}
	if _, apiErr := s.repoFactory.NewNotificationRepo().ResolveUserID(ctx, blockedAccountUserID); apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewParameterInvalidError("The user to block does not exist.", "account_user_id"))
		}
		return nil, tracing.Trace(span, apiErr)
	}
	blockID, apiErr := id.GenID(id.MessageBlockIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var block *domain.MessageBlock
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		created, apiErr := f.NewBlockRepo().Create(txCtx, blockID, accountID, callerAcus, blockedAccountUserID)
		if apiErr != nil {
			return apiErr
		}
		block = created
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeMessagingBlock,
			ResourceID:   created.ID,
			Changes:      audit.ComputeChanges(nil, created),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return block, nil
}

func (s *conversationSvcImpl) Unblock(ctx context.Context, blockedAccountUserID string) *apierror.APIError {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.unblock")
	defer span.End()

	_, callerAcus, _, apiErr := s.caller(ctx)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	// BlockRepo has no by-pair getter, so resolve the block's type id (for the delete event) from the caller's block list. A missing row means there is nothing to delete or audit.
	blocks, apiErr := s.repoFactory.NewBlockRepo().List(ctx, callerAcus)
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	var existing *domain.MessageBlock
	for _, b := range blocks {
		if b.BlockedAccountUserID == blockedAccountUserID {
			existing = b
			break
		}
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if apiErr := f.NewBlockRepo().Delete(txCtx, callerAcus, blockedAccountUserID); apiErr != nil {
			return apiErr
		}
		// No block existed for this pair: the delete was a no-op, so emit no event.
		if existing == nil {
			return nil
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionDelete,
			ResourceType: constants.ObjectTypeMessagingBlock,
			ResourceID:   existing.ID,
			Changes:      audit.ComputeChanges(existing, (*domain.MessageBlock)(nil)),
		})
	})
	if apiErr != nil {
		return tracing.Trace(span, apiErr)
	}
	return nil
}

func (s *conversationSvcImpl) ListBlocks(ctx context.Context) ([]*domain.MessageBlock, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.list_blocks")
	defer span.End()

	_, callerAcus, _, apiErr := s.caller(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewBlockRepo().List(ctx, callerAcus)
}

// ListContacts returns the caller's messaging directory. A customer relation actor never sees the individual staff of the vendor account — they get a single "support" contact. Internal actors get the active account_users in their account (including themselves, since a self-DM is allowed), filtered by a case-insensitive name substring.
func (s *conversationSvcImpl) ListContacts(ctx context.Context, query string) ([]*domain.MessagingContact, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.list_contacts")
	defer span.End()

	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || !identity.IsActorSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("Authentication is required."))
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if identity.IsRelationActor() {
		return []*domain.MessagingContact{{
			Type: domain.MessagingContactTypeSupport,
			Name: "Customer Service",
		}}, nil
	}

	return s.repoFactory.NewNotificationRepo().ListMessagingContacts(ctx, identity.Target.AccountID, query)
}

// ReportConversation persists a minimal abuse report against a conversation (optionally a specific message). Only an active participant of the conversation may report it.
func (s *conversationSvcImpl) ReportConversation(ctx context.Context, conversationID string, messageID *string, reason string) (*domain.MessageReport, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.report_conversation")
	defer span.End()

	if reason == "" {
		return nil, tracing.Trace(span, apierror.NewParameterMissingError("A reason is required.", "reason"))
	}

	// Participant gate (active members only) — supports internal and customer actors.
	rc, apiErr := s.resolveParticipant(ctx, conversationID, false)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Recovery-point idempotency: an abuse report mints a fresh id with no dedup key, so a retry would file a duplicate report.
	identity, _ := appctx.GetIdentityFromContext(ctx)
	idemKey, apiErr := upsertIdempotencyKey(ctx, s.repoFactory, identity)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	switch domain.RecoveryPoint(idemKey.RecoveryPoint) {
	case domain.RecoveryPointFinished:
		cached, err := idempotency.UnmarshalCachedResponse[domain.MessageReport](ctx, idemKey.ResponseCode, idemKey.ResponseBody)
		if err != nil {
			return nil, tracing.Trace(span, apierror.NewInternalError(err, "Issue unmarshalling cached response."))
		}
		return cached.Data, cached.Error
	case domain.RecoveryPointStarted:
		reportID, apiErr := id.GenID(id.MessageReportIDPrefix, nil)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
		// The reporter is the caller's account_user when internal; a customer relation actor has no account_user, so the report is attributed to their participant row instead.
		reporter := rc.senderAcus
		if reporter == "" {
			reporter = rc.part.ID
		}
		report := &domain.MessageReport{
			ID:                    reportID,
			AccountID:             rc.accountID,
			ConversationID:        conversationID,
			ReporterAccountUserID: reporter,
			Reason:                reason,
			CreatedAt:             time.Now(),
		}
		if messageID != nil {
			report.MessageID = *messageID
		}
		apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
			if cErr := f.NewMessageReportRepo().Create(txCtx, report); cErr != nil {
				return cErr
			}
			return cacheSuccessResponse(txCtx, f, idemKey.TypeID, report)
		})
		if apiErr != nil {
			return nil, tracing.Trace(span, cacheErrorResponse(ctx, s.repoFactory, idemKey.TypeID, apiErr))
		}
		return report, nil
	default:
		return nil, tracing.Trace(span, apierror.NewInvariantViolationError("Unexpected recovery point: "+idemKey.RecoveryPoint))
	}
}

// requireParticipant resolves the caller and loads their participant row for a conversation, returning a not-found (existence not leaked) when they are not a member.
func (s *conversationSvcImpl) requireParticipant(ctx context.Context, conversationID string) (string, string, *domain.ConversationParticipant, *apierror.APIError) {
	_, callerAcus, accountID, apiErr := s.caller(ctx)
	if apiErr != nil {
		return "", "", nil, apiErr
	}
	part, apiErr := s.repoFactory.NewParticipantRepo().Get(ctx, conversationID, callerAcus)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return "", "", nil, apierror.NewResourceNotFoundError("Conversation not found.")
		}
		return "", "", nil, apiErr
	}
	if part.Membership != string(constants.ParticipantMembershipActive) {
		return "", "", nil, apierror.NewResourceNotFoundError("Conversation not found.") // left/removed: no access
	}
	return callerAcus, accountID, part, nil
}

// requireMessagingAdmin authorizes an account-level messaging-admin operation (legal hold, redaction): an internal actor in the target account holding the given messaging permission.
// Customer relation actors are rejected. Returns the resolved account id.
func (s *conversationSvcImpl) requireMessagingAdmin(ctx context.Context, action types.Action) (string, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || !identity.IsActorSet() {
		return "", apierror.NewAuthenticationError("Authentication is required.")
	}
	if !identity.IsTargetAccountSet() {
		return "", apierror.NewAuthenticationError("The Augno-Account-ID header is required.")
	}
	if identity.IsRelationActor() {
		return "", apierror.NewAuthorizationError("This operation is not available to customer accounts.")
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainMessaging, action); apiErr != nil {
		return "", apiErr
	}
	return identity.Target.AccountID, nil
}

// loadConversationForAdmin returns a conversation plus its participants without requiring the caller to be a member (used by account-level admin operations). Unread is not caller-scoped here.
func (s *conversationSvcImpl) loadConversationForAdmin(ctx context.Context, conversationID, accountID string) (*domain.Conversation, *apierror.APIError) {
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, accountID)
	if apiErr != nil {
		return nil, apiErr
	}
	participants, apiErr := s.repoFactory.NewParticipantRepo().List(ctx, conversationID)
	if apiErr != nil {
		return nil, apiErr
	}
	conv.Participants = participants
	return conv, nil
}

func (s *conversationSvcImpl) SetLegalHold(ctx context.Context, conversationID string, hold bool) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.set_legal_hold")
	defer span.End()

	accountID, apiErr := s.requireMessagingAdmin(ctx, types.ActionUpdate)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	old, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		convRepo := f.NewConversationRepo()
		if apiErr := convRepo.SetLegalHold(txCtx, conversationID, accountID, hold); apiErr != nil {
			return apiErr
		}
		updated, apiErr := convRepo.GetByID(txCtx, conversationID, accountID)
		if apiErr != nil {
			return apiErr
		}
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeConversation,
			ResourceID:   updated.ID,
			Changes:      audit.ComputeChanges(old, updated),
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.loadConversationForAdmin(ctx, conversationID, accountID)
}

func (s *conversationSvcImpl) RedactConversation(ctx context.Context, conversationID string) (*domain.Conversation, *apierror.APIError) {
	ctx, span := conversationSvcTracer.Start(ctx, "service.conversation.redact")
	defer span.End()

	accountID, apiErr := s.requireMessagingAdmin(ctx, types.ActionDelete)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, accountID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if conv.LegalHold {
		return nil, tracing.Trace(span, apierror.NewValidationError("This conversation is under legal hold and cannot be redacted until the hold is cleared."))
	}

	// Delete attachment objects first (S3 before row) so a crash mid-redaction re-attempts the object delete rather than orphaning it. The message bodies are then cleared in a single transaction.
	attachments, apiErr := s.repoFactory.NewMessageAttachmentRepo().ListByConversation(ctx, conversationID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	for _, a := range attachments {
		if a.S3Key != nil && *a.S3Key != "" {
			if apiErr := s.objectStore.Delete(ctx, s.chatBucket, *a.S3Key); apiErr != nil {
				return nil, tracing.Trace(span, apiErr)
			}
		}
		if apiErr := s.repoFactory.NewMessageAttachmentRepo().DeleteByID(ctx, a.ID); apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		if _, apiErr := f.NewConversationRepo().RedactMessages(txCtx, conversationID); apiErr != nil {
			return apiErr
		}
		// Redaction clears message rows, not conversation audit-tagged fields, so the diff is empty; Metadata keeps the update event from being dropped.
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionUpdate,
			ResourceType: constants.ObjectTypeConversation,
			ResourceID:   conversationID,
			Metadata:     map[string]any{"redacted": true},
		})
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	s.fanoutConversationEvent(ctx, conversationID, accountID, "conversation.updated", "")
	return s.loadConversationForAdmin(ctx, conversationID, accountID)
}

// fanoutConversationEvent best-effort pushes a conversation/message event to every active participant's user topic and the conversation topic. Clients dedupe by id.
func (s *conversationSvcImpl) fanoutConversationEvent(ctx context.Context, conversationID, accountID, event, messageID string) {
	participants, apiErr := s.repoFactory.NewParticipantRepo().List(ctx, conversationID)
	if apiErr != nil {
		return
	}
	notifRepo := s.repoFactory.NewNotificationRepo()
	outbox := s.repoFactory.NewOutboxRepo()
	for _, p := range participants {
		if p.AccountUserID == nil || *p.AccountUserID == "" {
			continue
		}
		userID, apiErr := notifRepo.ResolveUserID(ctx, *p.AccountUserID)
		if apiErr != nil {
			continue
		}
		_ = enqueueRealtimeVia(ctx, outbox, messaging.RealtimeDeliveryData{
			AccountID:       accountID,
			RecipientUserID: userID,
			ConversationID:  conversationID,
			Event:           event,
			MessageID:       messageID,
		})
	}
}

// enforceDMBlock rejects sending in a direct message when either participant has blocked the other. No-op for non-DM conversations.
func (s *conversationSvcImpl) enforceDMBlock(ctx context.Context, conversationID, accountID, callerAcus string) *apierror.APIError {
	conv, apiErr := s.repoFactory.NewConversationRepo().GetByID(ctx, conversationID, accountID)
	if apiErr != nil {
		return apiErr
	}
	if conv.Type != string(constants.ConversationTypeDM) {
		return nil
	}
	participants, apiErr := s.repoFactory.NewParticipantRepo().List(ctx, conversationID)
	if apiErr != nil {
		return apiErr
	}
	blockRepo := s.repoFactory.NewBlockRepo()
	for _, p := range participants {
		if p.AccountUserID == nil || *p.AccountUserID == callerAcus {
			continue
		}
		blocked, apiErr := blockRepo.ExistsBetween(ctx, callerAcus, *p.AccountUserID)
		if apiErr != nil {
			return apiErr
		}
		if blocked {
			return apierror.NewAuthorizationError("You cannot send messages in this conversation.")
		}
	}
	return nil
}

// roleAllows reports whether role is one of the allowed roles.
func roleAllows(role string, allowed ...constants.ParticipantRole) bool {
	for _, a := range allowed {
		if role == string(a) {
			return true
		}
	}
	return false
}

// canManageAgentsInConversation reports whether the caller may add or remove agent participants. DMs and customer-facing cases allow any active participant; internal groups require owner or admin.
func canManageAgentsInConversation(conv *domain.Conversation, part *domain.ConversationParticipant) bool {
	if conv.Type == string(constants.ConversationTypeDM) {
		return true
	}
	if conv.Audience == string(constants.ConversationAudienceCustomer) {
		return true
	}
	return roleAllows(part.Role, constants.ParticipantRoleOwner, constants.ParticipantRoleAdmin)
}

// ── helpers ─────────────────────────────────────────────────────────

// buildDMKey produces the order-independent dedupe key for a DM participant pair.
func buildDMKey(a, b string) string {
	pair := []string{a, b}
	sort.Strings(pair)
	return pair[0] + ":" + pair[1]
}

func unreadFrom(nextSequence, lastReadSequence int64) int64 {
	if maxSeq := nextSequence - 1; maxSeq > lastReadSequence {
		return maxSeq - lastReadSequence
	}
	return 0
}

func truncatePreview(body string) string {
	if len(body) <= messagePreviewMaxLen {
		return body
	}
	return body[:messagePreviewMaxLen]
}

// messagePreview builds the conversation-list preview for a message: the truncated body, or an attachment placeholder when the message is attachment-only.
func messagePreview(body *string, attachmentCount int, hasLink bool) string {
	if body != nil {
		if p := truncatePreview(*body); p != "" {
			return p
		}
	}
	if attachmentCount > 0 {
		return "📎 Attachment"
	}
	if hasLink {
		return "🔗 Link"
	}
	return ""
}

func clampLimit(limit, def, max int32) int32 {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}

func encodeSeqCursor(sequence int64) string {
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(sequence, 10)))
}

func decodeSeqCursor(cursor *string) (*int64, *apierror.APIError) {
	if cursor == nil || *cursor == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(*cursor)
	if err != nil {
		return nil, apierror.NewParameterInvalidError("The cursor is invalid.", "cursor")
	}
	seq, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil {
		return nil, apierror.NewParameterInvalidError("The cursor is invalid.", "cursor")
	}
	return &seq, nil
}

// enqueueRealtimeVia writes a best-effort realtime-delivery event to the given outbox (which may be transaction-bound). The gateway WS consumer fans it out to the relevant Hub topic(s).
func enqueueRealtimeVia(ctx context.Context, outbox messaging.OutboxRepo, rt messaging.RealtimeDeliveryData) *apierror.APIError {
	payload, err := json.Marshal(rt)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal realtime delivery payload.")
	}
	msg := contracts.AmqpMessage{Data: payload}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}
	if _, err := outbox.Create(ctx, messaging.OutboxMessageInput{
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.NotificationEventDelivered),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationEventDelivered),
		Payload:     msg,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to enqueue realtime delivery.")
	}
	return nil
}

// enqueueAgentContinueRunVia writes an agent continue-run command to the given (transaction-bound)
// outbox. The agent-service consumes it and continues the run with the chat message as input.

// enqueueEmailVia writes a send-email command to the given (transaction-bound) outbox. The notification-service email consumer renders the template and dispatches via SES.
func enqueueEmailVia(ctx context.Context, outbox messaging.OutboxRepo, email messaging.EmailSendData) *apierror.APIError {
	payload, err := json.Marshal(email)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal email payload.")
	}
	msg := contracts.AmqpMessage{Data: payload}
	if identity, ok := appctx.GetIdentityFromContext(ctx); ok {
		msg.Identity = identity
	}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}
	if _, err := outbox.Create(ctx, messaging.OutboxMessageInput{
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.NotificationCmdSendEmail),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdSendEmail),
		Payload:     msg,
	}); err != nil {
		return apierror.NewInternalError(err, "Failed to enqueue chat-notification email.")
	}
	return nil
}
