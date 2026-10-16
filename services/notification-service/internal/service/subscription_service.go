package service

import (
	"context"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	"github.com/open-mrp/api/shared/audit"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/id"
	"github.com/open-mrp/api/shared/messaging"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/tracing"
)

const (
	defaultSubscriptionPageSize int32 = 25
	maxSubscriptionPageSize     int32 = 100
)

// subscriber is the caller's account user, refusing callers who have none (an API key can't follow records).
func (s *messagingSvcImpl) subscriber(ctx context.Context) (string, string, *apierror.APIError) {
	identity, apiErr := s.actor(ctx)
	if apiErr != nil {
		return "", "", apiErr
	}
	recipient, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return "", "", apiErr
	}
	if recipient == "" {
		return "", "", apierror.NewAuthorizationError("Subscriptions are only available to users with an account membership.")
	}
	return identity.Target.AccountID, recipient, nil
}

func (s *messagingSvcImpl) ListSubscriptions(ctx context.Context, input domain.ListSubscriptionsInput) (*domain.SubscriptionPage, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.list_subscriptions")
	defer span.End()

	recipient, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	page := &domain.SubscriptionPage{}
	if recipient == "" {
		return page, nil
	}
	limit := input.Limit
	if limit <= 0 {
		limit = defaultSubscriptionPageSize
	}
	if limit > maxSubscriptionPageSize {
		limit = maxSubscriptionPageSize
	}
	cursorAt, cursorID, apiErr := decodeCursor(input.Cursor)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	rows, apiErr := s.repoFactory.NewSubscriptionRepo().List(ctx, domain.SubscriptionListFilter{
		AccountUserID:   recipient,
		RecordIDs:       input.RecordIDs,
		RecordTypes:     input.RecordTypes,
		Statuses:        input.Statuses,
		Limit:           limit + 1,
		CursorCreatedAt: cursorAt,
		CursorID:        cursorID,
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if len(rows) > int(limit) {
		rows = rows[:limit]
		page.HasNextPage = true
		last := rows[len(rows)-1]
		next := encodeCursor(last.CreatedAt, last.ID)
		page.NextCursor = &next
	}
	page.Subscriptions = rows
	return page, nil
}

func (s *messagingSvcImpl) GetSubscription(ctx context.Context, subscriptionID string) (*domain.Subscription, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.get_subscription")
	defer span.End()
	recipient, apiErr := s.recipient(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	if recipient == "" {
		return nil, tracing.Trace(span, apierror.NewNotFoundError("Subscription not found."))
	}
	return s.repoFactory.NewSubscriptionRepo().Get(ctx, subscriptionID, recipient)
}

// FollowRecord subscribes the caller to a record. Following a record you already follow changes nothing; following
// one you unsubscribed from subscribes you again.
func (s *messagingSvcImpl) FollowRecord(ctx context.Context, recordType, recordID string) (*domain.Subscription, bool, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.follow_record")
	defer span.End()

	if !constants.RecordType(recordType).Followable() {
		return nil, false, tracing.Trace(span, apierror.NewInvalidFormatError("record.type", "Records of this type can't be followed."))
	}
	if recordID == "" {
		return nil, false, tracing.Trace(span, apierror.NewMissingFieldError("record.id", "A record ID is required."))
	}
	accountID, recipient, apiErr := s.subscriber(ctx)
	if apiErr != nil {
		return nil, false, tracing.Trace(span, apiErr)
	}

	var out *domain.Subscription
	created := false
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		repo := f.NewSubscriptionRepo()
		existing, apiErr := repo.GetByRecord(txCtx, recipient, recordID)
		if apiErr != nil && apiErr.Code != apierror.CodeResourceNotFound {
			return apiErr
		}
		if existing != nil {
			if existing.RecordType != recordType {
				return apierror.NewInvalidFormatError("record.type", "The record's type doesn't match its ID.")
			}
			if existing.Status == string(constants.RecordSubscriptionStatusSubscribed) {
				out = existing
				return nil
			}
			out, apiErr = s.setSubscriptionStatus(txCtx, f, existing, constants.RecordSubscriptionStatusSubscribed)
			return apiErr
		}
		subID, apiErr := id.GenID(id.SubscriptionIDPrefix, nil)
		if apiErr != nil {
			return apiErr
		}
		sub := &domain.Subscription{
			ID: subID, AccountID: accountID, AccountUserID: recipient, RecordType: recordType, RecordID: recordID,
			Reason: string(constants.SubscriptionReasonManual), Status: string(constants.RecordSubscriptionStatusSubscribed),
		}
		if _, apiErr := repo.Insert(txCtx, sub); apiErr != nil {
			return apiErr
		}
		out, apiErr = repo.Get(txCtx, subID, recipient)
		if apiErr != nil {
			return apiErr
		}
		created = true
		return audit.NewPublisher().Publish(txCtx, f.NewOutboxRepo(), audit.EventData{
			ServiceName:  domain.ServiceName,
			Action:       constants.AuditActionCreate,
			ResourceType: constants.ObjectTypeSubscription,
			ResourceID:   out.ID,
			Changes:      audit.ComputeChanges(nil, out),
		})
	})
	if apiErr != nil {
		return nil, false, tracing.Trace(span, apiErr)
	}
	return out, created, nil
}

// SetSubscriptionStatus subscribes or unsubscribes one of the caller's subscriptions. Setting the status it already
// has changes nothing.
func (s *messagingSvcImpl) SetSubscriptionStatus(ctx context.Context, subscriptionID, status string) (*domain.Subscription, *apierror.APIError) {
	ctx, span := messagingSvcTracer.Start(ctx, "service.messaging.set_subscription_status")
	defer span.End()
	if !constants.RecordSubscriptionStatus(status).IsValid() {
		return nil, tracing.Trace(span, apierror.NewInvalidFormatError("status", "The status is not supported."))
	}
	_, recipient, apiErr := s.subscriber(ctx)
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	var out *domain.Subscription
	apiErr = s.txManager.WithTx(ctx, func(txCtx context.Context, f domain.RepoFactory) *apierror.APIError {
		sub, apiErr := f.NewSubscriptionRepo().Get(txCtx, subscriptionID, recipient)
		if apiErr != nil {
			return apiErr
		}
		if sub.Status == status {
			out = sub
			return nil
		}
		out, apiErr = s.setSubscriptionStatus(txCtx, f, sub, constants.RecordSubscriptionStatus(status))
		return apiErr
	})
	if apiErr != nil {
		return nil, tracing.Trace(span, apiErr)
	}
	return out, nil
}

// setSubscriptionStatus moves a subscription to status. Subscribing again by hand makes the reason manual.
func (s *messagingSvcImpl) setSubscriptionStatus(ctx context.Context, f domain.RepoFactory, before *domain.Subscription, status constants.RecordSubscriptionStatus) (*domain.Subscription, *apierror.APIError) {
	reason := before.Reason
	if status == constants.RecordSubscriptionStatusSubscribed {
		reason = string(constants.SubscriptionReasonManual)
	}
	repo := f.NewSubscriptionRepo()
	if apiErr := repo.SetStatus(ctx, before.ID, before.AccountUserID, string(status), reason); apiErr != nil {
		return nil, apiErr
	}
	after, apiErr := repo.Get(ctx, before.ID, before.AccountUserID)
	if apiErr != nil {
		return nil, apiErr
	}
	return after, audit.NewPublisher().Publish(ctx, f.NewOutboxRepo(), audit.EventData{
		ServiceName:  domain.ServiceName,
		Action:       constants.AuditActionUpdate,
		ResourceType: constants.ObjectTypeSubscription,
		ResourceID:   after.ID,
		Changes:      audit.ComputeChanges(before, after),
	})
}

// applySubscriptionIntents subscribes people to a record automatically, each only when their auto-subscribe setting
// includes the reason. A person who already has a row for the record (subscribed or not) keeps it.
func (s *messagingSvcImpl) applySubscriptionIntents(ctx context.Context, f domain.RepoFactory, cache *settingsCache, data messaging.AlertFanoutData) *apierror.APIError {
	if data.Record == nil || len(data.Subscribe) == 0 || !constants.RecordType(data.Record.Type).Followable() {
		return nil
	}
	repo := f.NewSubscriptionRepo()
	for _, intent := range data.Subscribe {
		reason := constants.SubscriptionReason(intent.Reason)
		if !reason.IsValid() {
			continue
		}
		var accountUsers, users []string
		if intent.AccountUserID != "" {
			accountUsers = []string{intent.AccountUserID}
		} else if intent.UserID != "" {
			users = []string{intent.UserID}
		}
		for _, rc := range s.resolveRecipients(ctx, data.AccountID, accountUsers, users) {
			if !cache.get(ctx, rc.accountUserID).layers.AutoSubscribes(reason) {
				continue
			}
			subID, apiErr := id.GenID(id.SubscriptionIDPrefix, nil)
			if apiErr != nil {
				return apiErr
			}
			if _, apiErr := repo.Insert(ctx, &domain.Subscription{
				ID: subID, AccountID: data.AccountID, AccountUserID: rc.accountUserID, RecordType: data.Record.Type,
				RecordID: data.Record.ID, Reason: string(reason), Status: string(constants.RecordSubscriptionStatusSubscribed),
			}); apiErr != nil {
				return apiErr
			}
		}
	}
	return nil
}

// audienceRecipients is everyone subscribed to the audience's records, minus the people it excludes. When nobody has
// ever had a subscription row for the first record (one made before subscriptions existed), the fallback recipients
// are used instead.
func (s *messagingSvcImpl) audienceRecipients(ctx context.Context, f domain.RepoFactory, accountID string, a *messaging.SubscriberAudience) ([]recipientRef, *apierror.APIError) {
	repo := f.NewSubscriptionRepo()
	var accountUsers []string
	for _, r := range a.Records {
		ids, apiErr := repo.Subscribers(ctx, accountID, r.Type, r.ID)
		if apiErr != nil {
			return nil, apiErr
		}
		accountUsers = append(accountUsers, ids...)
	}
	var users []string
	if len(a.Records) > 0 && len(accountUsers) == 0 {
		has, apiErr := repo.HasAny(ctx, accountID, a.Records[0].Type, a.Records[0].ID)
		if apiErr != nil {
			return nil, apiErr
		}
		if !has {
			accountUsers, users = a.FallbackAccountUserIDs, a.FallbackUserIDs
		}
	}
	excluded := map[string]bool{}
	for _, rc := range s.resolveRecipients(ctx, accountID, a.ExcludeAccountUserIDs, a.ExcludeUserIDs) {
		excluded[rc.accountUserID] = true
	}
	var out []recipientRef
	for _, rc := range s.resolveRecipients(ctx, accountID, accountUsers, users) {
		if !excluded[rc.accountUserID] {
			out = append(out, rc)
		}
	}
	return out, nil
}
