package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/notification-service/internal/domain"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/id"
	"github.com/augno/api/shared/messaging"
	"github.com/augno/api/shared/tracing"
)

var messagingSvcTracer = tracing.GetTracer("notification-service.messaging_service")

const (
	defaultNotificationPageSize int32 = 50
	maxNotificationPageSize     int32 = 100
)

type messagingSvcImpl struct {
	repoFactory domain.RepoFactory
}

// NewMessagingSvc constructs the in-app notification (bell) service.
func NewMessagingSvc(repoFactory domain.RepoFactory) domain.MessagingSvc {
	return &messagingSvcImpl{repoFactory: repoFactory}
}

// actor resolves and validates the calling identity.
func (s *messagingSvcImpl) actor(ctx context.Context) (*types.Identity, *apierror.APIError) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || !identity.IsActorSet() {
		return nil, apierror.NewAuthenticationError("Authentication is required.")
	}
	return identity, nil
}

// recipient resolves the account_user_id the request is scoped to. identity.Actor.ID is the user id (us_), not the account_user id (acus_) the notification feed is keyed by, so we resolve it from (user_id, target account_id) via the unique (user_id, account_id) key.
func (s *messagingSvcImpl) recipient(ctx context.Context) (string, *apierror.APIError) {
	identity, apiErr := s.actor(ctx)
	if apiErr != nil {
		return "", apiErr
	}
	if !identity.IsTargetAccountSet() {
		return "", apierror.NewAuthenticationError("The Augno-Account-ID header is required.")
	}
	recipientID, apiErr := s.repoFactory.NewNotificationRepo().ResolveAccountUserID(ctx, identity.Actor.ID, identity.Target.AccountID)
	if apiErr != nil {
		// Actors without an account_user in the target account (e.g. API keys) have no personal feed. Return an empty recipient so reads yield an empty feed / zero counts rather than a 404 — a list endpoint should not 404 for an authed caller.
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return "", nil
		}
		return "", apiErr
	}
	return recipientID, nil
}

func (s *messagingSvcImpl) SendNotification(ctx context.Context, input domain.SendNotificationInput) (int64, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.send_notification")
	defer span.End()

	identity, apiErr := s.actor(ctx)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	if apiErr := identity.CheckHasPermission(types.PermissionDomainAlerts, types.ActionCreate); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return 0, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}

	if input.Title == "" {
		return 0, tracing.Trace(span, apierror.NewParameterMissingError("A title is required.", "title"))
	}
	if input.TargetID == "" {
		return 0, tracing.Trace(span, apierror.NewParameterMissingError("A target is required.", "target"))
	}

	switch constants.NotificationTargetType(input.TargetType) {
	case constants.NotificationTargetTypeAccount:
		// An account target is stored once as an account-scoped announcement (with per-user receipts), not per-user rows, so it scales to every user in the account. An admin may only broadcast to the account they are acting in.
		if input.TargetID != identity.Target.AccountID {
			return 0, tracing.Trace(span, apierror.NewParameterInvalidError("Broadcasts may only target the current account.", "target.id"))
		}
		if apiErr := s.createBroadcastAnnouncement(ctx, identity, input); apiErr != nil {
			return 0, tracing.Trace(span, apiErr)
		}
		return 1, nil
	case constants.NotificationTargetTypeAccountUser:
		// Fall through to the per-user fan-out below.
	default:
		return 0, tracing.Trace(span, apierror.NewParameterInvalidError("The target type is not supported.", "target.type"))
	}

	data := messaging.AlertFanoutData{
		AccountID:               identity.Target.AccountID,
		Category:                input.Category,
		Kind:                    "alert",
		Title:                   input.Title,
		TemplateKey:             ptrToStr(input.TemplateKey),
		TemplateParams:          input.TemplateParams,
		LinkResourceType:        ptrToStr(input.LinkResourceType),
		LinkResourceID:          ptrToStr(input.LinkResourceID),
		Priority:                ptrToStr(input.Priority),
		RecipientAccountUserIDs: []string{input.TargetID},
	}
	if input.Body != nil {
		data.Body = *input.Body
	}
	// Attribute the notification to the calling actor, derived from the authenticated identity rather than client-supplied fields. A system/unattributed send leaves the sender empty (the recipient sees no sender).
	s.applySenderFromIdentity(ctx, &data, identity)

	if apiErr := s.enqueueFanout(ctx, identity, data); apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	return int64(len(data.RecipientAccountUserIDs)), nil
}

// applySenderFromIdentity attributes the notification to the calling actor. The sender id for a user is their account_user id (the form the Sender resource and feed filters use), resolved from the user id; api-key and agent actors are attributed by their actor id. Any other actor (or an unresolvable user) yields an unattributed (system) notification.
func (s *messagingSvcImpl) applySenderFromIdentity(ctx context.Context, data *messaging.AlertFanoutData, identity *types.Identity) {
	if identity == nil || identity.Actor == nil {
		return
	}
	switch identity.Type {
	case types.IdentityActorTypeUser:
		accountUserID, apiErr := s.repoFactory.NewNotificationRepo().ResolveAccountUserID(ctx, identity.Actor.ID, data.AccountID)
		if apiErr != nil {
			return // unresolvable membership → leave unattributed rather than mis-attribute
		}
		data.SenderType = string(constants.NotificationSenderTypeUser)
		data.SenderID = accountUserID
	case types.IdentityActorTypeAPIKey:
		data.SenderType = string(constants.NotificationSenderTypeAPIKey)
		data.SenderID = identity.Actor.ID
	case types.IdentityActorTypeAgent:
		data.SenderType = string(constants.NotificationSenderTypeAgent)
		data.SenderID = identity.Actor.ID
	default:
		return
	}
	if identity.Actor.Name != nil {
		data.SenderName = *identity.Actor.Name
	}
}

// enqueueFanout writes a notification.cmd.fanout intent to the outbox. The enqueuer publishes it and the fan-out consumer (FanOut) materializes the rows.
func (s *messagingSvcImpl) enqueueFanout(ctx context.Context, identity *types.Identity, data messaging.AlertFanoutData) *apierror.APIError {
	payload, err := json.Marshal(data)
	if err != nil {
		return apierror.NewInternalError(err, "Failed to marshal fan-out payload.")
	}

	msg := contracts.AmqpMessage{Data: payload, Identity: identity}
	if requestID, ok := appctx.GetRequestID(ctx); ok {
		msg.RequestID = requestID
	}

	_, err = s.repoFactory.NewOutboxRepo().Create(ctx, messaging.OutboxMessageInput{
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.NotificationCmdFanout),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationCmdFanout),
		Payload:     msg,
	})
	if err != nil {
		return apierror.NewInternalError(err, "Failed to enqueue notification fan-out.")
	}
	return nil
}

func (s *messagingSvcImpl) FanOut(ctx context.Context, dedupeSeed string, data messaging.AlertFanoutData) *apierror.APIError {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.fan_out")
	defer span.End()

	if data.Broadcast {
		// Broadcast-to-all is served by announcements; ignore here to avoid mass row writes.
		return nil
	}

	notifRepo := s.repoFactory.NewNotificationRepo()

	// Resolve the recipient set to (account_user_id, user_id) pairs. The notification row is keyed by account_user id; the realtime push targets the WS user-topic (user id). Producers may supply either form: RecipientAccountUserIDs (admin sends) or RecipientUserIDs (e.g. agent-service, which holds the triggering user id, not the per-account account_user id).
	recipients := s.resolveRecipients(ctx, data.AccountID, data.RecipientAccountUserIDs, data.RecipientUserIDs)
	if len(recipients) == 0 {
		return nil
	}

	notifications := make([]*domain.Notification, 0, len(recipients))
	userIDByNotificationID := make(map[string]string, len(recipients))
	for _, rc := range recipients {
		n := &domain.Notification{
			ID:                     deterministicNotificationID(dedupeSeed, rc.accountUserID),
			AccountID:              data.AccountID,
			RecipientAccountUserID: rc.accountUserID,
			Category:               data.Category,
			Title:                  data.Title,
			Body:                   strPtrIfNotEmpty(data.Body),
			TemplateKey:            strPtrIfNotEmpty(data.TemplateKey),
			TemplateParams:         data.TemplateParams,
			LinkResourceType:       strPtrIfNotEmpty(data.LinkResourceType),
			LinkResourceID:         strPtrIfNotEmpty(data.LinkResourceID),
			SenderType:             strPtrIfNotEmpty(data.SenderType),
			SenderID:               strPtrIfNotEmpty(data.SenderID),
			SenderName:             strPtrIfNotEmpty(data.SenderName),
			Priority:               data.Priority,
		}
		notifications = append(notifications, n)
		userIDByNotificationID[n.ID] = rc.userID
	}

	if apiErr := notifRepo.CreateBatch(ctx, notifications); apiErr != nil {
		return tracing.Trace(span, apiErr)
	}

	// Emit a best-effort realtime push per recipient. Duplicate pushes on redelivery are harmless — the client dedupes by notification id, and the persisted row is authoritative.
	for _, n := range notifications {
		if apiErr := s.publishRealtime(ctx, n, userIDByNotificationID[n.ID]); apiErr != nil {
			return tracing.Trace(span, apiErr)
		}
	}
	return nil
}

// recipientRef pairs the account_user id a notification row is keyed by with the user id its realtime push targets (the WS user-topic).
type recipientRef struct {
	accountUserID string
	userID        string
}

// resolveRecipients normalizes the two producer-supplied recipient forms into deduped (account_user_id, user_id) pairs, scoped to accountID. Entries that don't resolve (e.g. a user with no membership in this account) are skipped — fan-out is best-effort per recipient.
func (s *messagingSvcImpl) resolveRecipients(ctx context.Context, accountID string, accountUserIDs, userIDs []string) []recipientRef {
	repo := s.repoFactory.NewNotificationRepo()
	seen := make(map[string]struct{})
	refs := make([]recipientRef, 0, len(accountUserIDs)+len(userIDs))

	for _, accountUserID := range accountUserIDs {
		if accountUserID == "" {
			continue
		}
		if _, ok := seen[accountUserID]; ok {
			continue
		}
		userID, apiErr := repo.ResolveUserID(ctx, accountUserID)
		if apiErr != nil {
			continue // unknown account_user — skip; the row would still persist but never push
		}
		seen[accountUserID] = struct{}{}
		refs = append(refs, recipientRef{accountUserID: accountUserID, userID: userID})
	}

	for _, userID := range userIDs {
		if userID == "" {
			continue
		}
		accountUserID, apiErr := repo.ResolveAccountUserID(ctx, userID, accountID)
		if apiErr != nil {
			continue // user has no membership in this account — skip
		}
		if _, ok := seen[accountUserID]; ok {
			continue
		}
		seen[accountUserID] = struct{}{}
		refs = append(refs, recipientRef{accountUserID: accountUserID, userID: userID})
	}

	return refs
}

func (s *messagingSvcImpl) publishRealtime(ctx context.Context, n *domain.Notification, recipientUserID string) *apierror.APIError {
	return s.enqueueRealtime(ctx, messaging.RealtimeDeliveryData{
		AccountID:              n.AccountID,
		RecipientUserID:        recipientUserID,
		RecipientAccountUserID: n.RecipientAccountUserID,
		Event:                  "notification.created",
		NotificationID:         n.ID,
	})
}

// enqueueRealtime writes a best-effort realtime-delivery event to the outbox. The gateway WS consumer fans it out to the relevant Hub topic(s); persisted rows remain the source of truth, so callers may treat a failure here as non-fatal.
func (s *messagingSvcImpl) enqueueRealtime(ctx context.Context, rt messaging.RealtimeDeliveryData) *apierror.APIError {
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

	_, err = s.repoFactory.NewOutboxRepo().Create(ctx, messaging.OutboxMessageInput{
		ServiceName: domain.ServiceName,
		MessageType: string(contracts.NotificationEventDelivered),
		Destination: messaging.ApplicationExchange,
		RoutingKey:  string(contracts.NotificationEventDelivered),
		Payload:     msg,
	})
	if err != nil {
		return apierror.NewInternalError(err, "Failed to enqueue realtime delivery.")
	}
	return nil
}

// publishUnreadChanged emits a best-effort unread.changed event to the caller's user topic so the bell badge syncs across the caller's other tabs/devices (and, via the cross-account hint, their connections in other accounts).
func (s *messagingSvcImpl) publishUnreadChanged(ctx context.Context) {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || identity.Actor == nil || identity.Actor.ID == "" || !identity.IsTargetAccountSet() {
		return
	}
	_ = s.enqueueRealtime(ctx, messaging.RealtimeDeliveryData{
		AccountID:       identity.Target.AccountID,
		RecipientUserID: identity.Actor.ID,
		Event:           "unread.changed",
	})
}

func (s *messagingSvcImpl) ListNotifications(ctx context.Context, input domain.ListNotificationsInput) (*domain.NotificationPage, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.list_notifications")
	defer span.End()

	recipient, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultNotificationPageSize
	}
	if limit > maxNotificationPageSize {
		limit = maxNotificationPageSize
	}

	cursorCreatedAt, cursorID, apiErr := decodeCursor(input.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Fetch one extra row to determine whether a next page exists.
	rows, apiErr := s.repoFactory.NewNotificationRepo().List(ctx, domain.NotificationListFilter{
		RecipientAccountUserID: recipient,
		Category:               input.Category,
		Status:                 input.Status,
		Search:                 input.Search,
		SenderIDs:              input.SenderIDs,
		SenderTypes:            input.SenderTypes,
		Limit:                  limit + 1,
		CursorCreatedAt:        cursorCreatedAt,
		CursorID:               cursorID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	page := &domain.NotificationPage{}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		page.HasNextPage = true
		last := rows[len(rows)-1]
		next := encodeCursor(last.CreatedAt, last.ID)
		page.NextCursor = &next
	}
	page.Notifications = rows
	return page, nil
}

func (s *messagingSvcImpl) GetNotification(ctx context.Context, notificationID string) (*domain.Notification, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.get_notification")
	defer span.End()

	recipient, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewNotificationRepo().GetByID(ctx, notificationID, recipient)
}

func (s *messagingSvcImpl) GetUnreadCount(ctx context.Context) (*domain.UnreadCounts, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.get_unread_count")
	defer span.End()

	recipient, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	count, apiErr := s.repoFactory.NewNotificationRepo().CountUnseen(ctx, recipient)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// The bell total also surfaces broadcast announcements the caller hasn't seen. These are kept out of the per-user Notifications sub-count (which mark-all-seen clears) and folded into Total only, since announcements are dismissed individually via their own receipts.
	var announceCount int64
	if accountID := s.accountID(ctx); accountID != nil {
		announceCount, apiErr = s.repoFactory.NewAnnouncementRepo().CountUnseen(ctx, recipient, accountID)
		if apiErr != nil {
			return nil, tracing.Trace(span, apiErr)
		}
	}

	// Conversations is 0 until chat ships (Phase 2).
	return &domain.UnreadCounts{Notifications: count, Conversations: 0, Total: count + announceCount}, nil
}

func (s *messagingSvcImpl) GetUnreadSummary(ctx context.Context) (*domain.UnreadSummary, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.get_unread_summary")
	defer span.End()

	identity, apiErr := s.actor(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	userID := identity.Actor.ID
	notifCounts, apiErr := s.repoFactory.NewNotificationRepo().CountUnseenByUserAccounts(ctx, userID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	announceCounts, apiErr := s.repoFactory.NewAnnouncementRepo().CountUnseenByUserAccounts(ctx, userID)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	// Merge per-account notification and announcement tallies.
	byAccount := make(map[string]int64)
	for _, c := range notifCounts {
		byAccount[c.AccountID] += c.Unread
	}
	for _, c := range announceCounts {
		byAccount[c.AccountID] += c.Unread
	}

	summary := &domain.UnreadSummary{Accounts: make([]domain.AccountUnread, 0, len(byAccount))}
	for accountID, unread := range byAccount {
		summary.Accounts = append(summary.Accounts, domain.AccountUnread{AccountID: accountID, Unread: unread})
		summary.Total += unread
	}
	return summary, nil
}

// accountID returns the caller's target account id, or nil when unauthenticated/unset.
func (s *messagingSvcImpl) accountID(ctx context.Context) *string {
	identity, ok := appctx.GetIdentityFromContext(ctx)
	if !ok || identity == nil || !identity.IsTargetAccountSet() {
		return nil
	}
	return &identity.Target.AccountID
}

func (s *messagingSvcImpl) MarkSeen(ctx context.Context, notificationID string) (*domain.Notification, *apierror.APIError) {
	return s.mark(ctx, "mark_seen", func(repo domain.NotificationRepo, recipient string) (*domain.Notification, *apierror.APIError) {
		return repo.MarkSeen(ctx, notificationID, recipient)
	})
}

func (s *messagingSvcImpl) MarkRead(ctx context.Context, notificationID string) (*domain.Notification, *apierror.APIError) {
	return s.mark(ctx, "mark_read", func(repo domain.NotificationRepo, recipient string) (*domain.Notification, *apierror.APIError) {
		return repo.MarkRead(ctx, notificationID, recipient)
	})
}

func (s *messagingSvcImpl) MarkDismissed(ctx context.Context, notificationID string) (*domain.Notification, *apierror.APIError) {
	return s.mark(ctx, "mark_dismissed", func(repo domain.NotificationRepo, recipient string) (*domain.Notification, *apierror.APIError) {
		return repo.MarkDismissed(ctx, notificationID, recipient)
	})
}

func (s *messagingSvcImpl) mark(ctx context.Context, op string, fn func(domain.NotificationRepo, string) (*domain.Notification, *apierror.APIError)) (*domain.Notification, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging."+op)
	defer span.End()

	recipient, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	result, apiErr := fn(s.repoFactory.NewNotificationRepo(), recipient)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	s.publishUnreadChanged(ctx)
	return result, nil
}

func (s *messagingSvcImpl) MarkAllSeen(ctx context.Context) (int64, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.mark_all_seen")
	defer span.End()

	recipient, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	updated, apiErr := s.repoFactory.NewNotificationRepo().MarkAllSeen(ctx, recipient)
	if apiErr != nil {
		return 0, tracing.Trace(span, apiErr)
	}
	s.publishUnreadChanged(ctx)
	return updated, nil
}

// createBroadcastAnnouncement stores a broadcast as an account-scoped announcement. The announcement is read by every user in the account via per-user receipts, so a single row serves the whole account regardless of size.
func (s *messagingSvcImpl) createBroadcastAnnouncement(ctx context.Context, identity *types.Identity, input domain.SendNotificationInput) *apierror.APIError {
	announcementID, apiErr := id.GenID(id.AnnouncementIDPrefix, nil)
	if apiErr != nil {
		return apiErr
	}

	accountID := identity.Target.AccountID
	createdBy := identity.Actor.ID
	create := &domain.CreateAnnouncementInput{
		Scope:            string(constants.AnnouncementScopeAccount),
		AccountID:        &accountID,
		Category:         input.Category,
		Title:            input.Title,
		Body:             input.Body,
		TemplateKey:      input.TemplateKey,
		TemplateParams:   input.TemplateParams,
		LinkResourceType: input.LinkResourceType,
		LinkResourceID:   input.LinkResourceID,
		Priority:         ptrToStr(input.Priority),
		PublishAt:        time.Now(),
		CreatedBy:        &createdBy,
	}
	if apiErr := s.repoFactory.NewAnnouncementRepo().Create(ctx, announcementID, create); apiErr != nil {
		return apiErr
	}

	// Best-effort live push to every connected user in the account. The persisted announcement is the source of truth, so a failed push (users still see it on next poll/reconnect) must not fail the send.
	_ = s.enqueueRealtime(ctx, messaging.RealtimeDeliveryData{
		AccountID:      accountID,
		AnnouncementID: announcementID,
		Event:          "announcement.created",
	})
	return nil
}

func (s *messagingSvcImpl) ListAnnouncements(ctx context.Context, input domain.ListAnnouncementsInput) (*domain.AnnouncementPage, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.list_announcements")
	defer span.End()

	recipient, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	limit := input.Limit
	if limit <= 0 {
		limit = defaultNotificationPageSize
	}
	if limit > maxNotificationPageSize {
		limit = maxNotificationPageSize
	}

	cursorPublishAt, cursorID, apiErr := decodeCursor(input.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	rows, apiErr := s.repoFactory.NewAnnouncementRepo().ListActive(ctx, domain.AnnouncementListFilter{
		AccountUserID:   recipient,
		AccountID:       s.accountID(ctx),
		Limit:           limit + 1,
		CursorPublishAt: cursorPublishAt,
		CursorID:        cursorID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}

	page := &domain.AnnouncementPage{}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		page.HasNextPage = true
		last := rows[len(rows)-1]
		next := encodeCursor(last.PublishAt, last.ID)
		page.NextCursor = &next
	}
	page.Announcements = rows
	return page, nil
}

func (s *messagingSvcImpl) GetAnnouncement(ctx context.Context, announcementID string) (*domain.Announcement, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.get_announcement")
	defer span.End()

	recipient, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return s.repoFactory.NewAnnouncementRepo().GetActiveByID(ctx, announcementID, recipient, s.accountID(ctx))
}

func (s *messagingSvcImpl) MarkAnnouncementSeen(ctx context.Context, announcementID string) (*domain.Announcement, *apierror.APIError) {
	return s.markAnnouncement(ctx, "mark_announcement_seen", announcementID, func(repo domain.AnnouncementRepo, id, recipient string) *apierror.APIError {
		return repo.MarkSeen(ctx, id, recipient)
	})
}

func (s *messagingSvcImpl) MarkAnnouncementRead(ctx context.Context, announcementID string) (*domain.Announcement, *apierror.APIError) {
	return s.markAnnouncement(ctx, "mark_announcement_read", announcementID, func(repo domain.AnnouncementRepo, id, recipient string) *apierror.APIError {
		return repo.MarkRead(ctx, id, recipient)
	})
}

func (s *messagingSvcImpl) MarkAnnouncementDismissed(ctx context.Context, announcementID string) (*domain.Announcement, *apierror.APIError) {
	return s.markAnnouncement(ctx, "mark_announcement_dismissed", announcementID, func(repo domain.AnnouncementRepo, id, recipient string) *apierror.APIError {
		return repo.MarkDismissed(ctx, id, recipient)
	})
}

func (s *messagingSvcImpl) markAnnouncement(ctx context.Context, op, announcementID string, fn func(domain.AnnouncementRepo, string, string) *apierror.APIError) (*domain.Announcement, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging."+op)
	defer span.End()

	recipient, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if recipient == "" {
		// No account_user (e.g. an API key) → no receipt to write; nothing to mark.
		return nil, tracing.Trace(span, apierror.NewResourceNotFoundError("Announcement not found."))
	}

	repo := s.repoFactory.NewAnnouncementRepo()
	accountID := s.accountID(ctx)
	// Confirm the announcement is visible to the caller before writing a receipt (404 otherwise).
	if _, apiErr := repo.GetActiveByID(ctx, announcementID, recipient, accountID); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := fn(repo, announcementID, recipient); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	s.publishUnreadChanged(ctx)
	return repo.GetActiveByID(ctx, announcementID, recipient, accountID)
}

// deterministicNotificationID derives a stable id from (source message id, recipient) so the fan-out handler is idempotent across redelivery (re-inserts collide on the PK and are skipped).
func deterministicNotificationID(seed, recipient string) string {
	sum := sha256.Sum256([]byte(seed + "|" + recipient))
	return string(id.NotificationIDPrefix) + "_" + hex.EncodeToString(sum[:])[:22]
}

func encodeCursor(createdAt time.Time, rowID string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(createdAt.Format(time.RFC3339Nano) + "|" + rowID))
}

func decodeCursor(cursor *string) (*time.Time, *string, *apierror.APIError) {
	if cursor == nil || *cursor == "" {
		return nil, nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(*cursor)
	if err != nil {
		return nil, nil, apierror.NewParameterInvalidError("The cursor is invalid.", "cursor")
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return nil, nil, apierror.NewParameterInvalidError("The cursor is invalid.", "cursor")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return nil, nil, apierror.NewParameterInvalidError("The cursor is invalid.", "cursor")
	}
	return &createdAt, &parts[1], nil
}

func ptrToStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func strPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *messagingSvcImpl) ListNotificationPreferences(ctx context.Context) ([]*domain.NotificationPreference, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.list_notification_preferences")
	defer span.End()

	recipientID, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if recipientID == "" {
		// Actors without an account_user (e.g. API keys) have no personal preferences.
		return []*domain.NotificationPreference{}, nil
	}
	return s.repoFactory.NewNotificationPreferenceRepo().List(ctx, recipientID)
}

func (s *messagingSvcImpl) UpsertNotificationPreference(ctx context.Context, input domain.UpsertNotificationPreferenceInput) (*domain.NotificationPreference, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.upsert_notification_preference")
	defer span.End()

	identity, apiErr := s.actor(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if !identity.IsTargetAccountSet() {
		return nil, tracing.Trace(span, apierror.NewAuthenticationError("The Augno-Account-ID header is required."))
	}
	accountID := identity.Target.AccountID
	recipientID, apiErr := s.repoFactory.NewNotificationRepo().ResolveAccountUserID(ctx, identity.Actor.ID, accountID)
	if apiErr != nil {
		if apiErr.Code == apierror.ErrorCodeResourceNotFound {
			return nil, tracing.Trace(span, apierror.NewAuthorizationError("Notification preferences are only available to users with an account membership."))
		}
		return nil, tracing.Trace(span, apiErr)
	}

	if input.Digest == "" {
		input.Digest = string(constants.NotificationDigestInstant)
	}
	if !constants.NotificationDigest(input.Digest).IsValid() {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("An invalid digest was supplied.", "digest"))
	}
	if input.Category != "" && !constants.NotificationCategory(input.Category).IsValid() {
		return nil, tracing.Trace(span, apierror.NewParameterInvalidError("An invalid category was supplied.", "category"))
	}

	prefRepo := s.repoFactory.NewNotificationPreferenceRepo()
	prefID, apiErr := id.GenID(id.NotificationPreferenceIDPrefix, nil)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if apiErr := prefRepo.Upsert(ctx, prefID, accountID, recipientID, &input); apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return prefRepo.GetByUserCategory(ctx, recipientID, input.Category)
}
