package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/services/notification-service/internal/domain"
	factorymock "github.com/augno/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/notification-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/appctx"
	"github.com/augno/api/shared/contracts"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/messaging"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// fakeOutboxRepo is a hand-written stand-in for messaging.OutboxRepo (no generated mock,
// since it's a shared interface). It records every Create input for assertion.
type fakeOutboxRepo struct {
	inputs []messaging.OutboxMessageInput
	err    error
}

func (f *fakeOutboxRepo) Create(_ context.Context, input messaging.OutboxMessageInput) (int64, error) {
	f.inputs = append(f.inputs, input)
	return int64(len(f.inputs)), f.err
}

const (
	testUserID    = "us_test1234"
	testAccountID = "ac_test1234"
	testRecipient = "acus_test1234"
)

// identityCtx builds a context carrying an internal user identity with the given permissions.
func identityCtx(perms ...string) context.Context {
	p := map[string]bool{}
	for _, perm := range perms {
		p[perm] = true
	}
	acct := testAccountID
	identity := &types.Identity{
		Type:   types.IdentityActorTypeUser,
		Target: &types.IdentityTarget{AccountID: testAccountID},
		Actor: &types.IdentityActor{
			RelationType: types.IdentityRelationTypeInternal,
			ID:           testUserID,
			AccountID:    &acct,
			Permissions:  p,
		},
	}
	return appctx.WithIdentity(context.Background(), identity)
}

// newSvc wires the messaging service with a mock factory returning the given repos.
func newSvc(t *testing.T, notifRepo domain.NotificationRepo, outboxRepo messaging.OutboxRepo) domain.MessagingSvc {
	t.Helper()
	ctrl := gomock.NewController(t)
	factory := factorymock.NewMockRepoFactory(ctrl)
	if notifRepo != nil {
		factory.EXPECT().NewNotificationRepo().Return(notifRepo).AnyTimes()
	}
	if outboxRepo != nil {
		factory.EXPECT().NewOutboxRepo().Return(outboxRepo).AnyTimes()
	}
	return NewMessagingSvc(factory)
}

// ── Cursor encode/decode ───────────────────────────────────────────

func TestCursor_RoundTrip(t *testing.T) {
	created := time.Date(2026, 6, 20, 8, 30, 0, 123456789, time.UTC)
	id := "nf_abc123"

	cursor := encodeCursor(created, id)
	gotTime, gotID, apiErr := decodeCursor(&cursor)
	require.Nil(t, apiErr)
	require.NotNil(t, gotTime)
	require.NotNil(t, gotID)
	assert.True(t, created.Equal(*gotTime), "round-tripped time should match (got %s)", gotTime)
	assert.Equal(t, id, *gotID)
}

func TestCursor_NilAndEmpty(t *testing.T) {
	gotTime, gotID, apiErr := decodeCursor(nil)
	require.Nil(t, apiErr)
	assert.Nil(t, gotTime)
	assert.Nil(t, gotID)

	empty := ""
	gotTime, gotID, apiErr = decodeCursor(&empty)
	require.Nil(t, apiErr)
	assert.Nil(t, gotTime)
	assert.Nil(t, gotID)
}

func TestCursor_Malformed(t *testing.T) {
	cases := map[string]string{
		"not base64":        "!!!not-base64!!!",
		"missing separator": base64Raw("no-pipe-here"),
		"bad timestamp":     base64Raw("not-a-time|nf_x"),
	}
	for name, cursor := range cases {
		t.Run(name, func(t *testing.T) {
			c := cursor
			_, _, apiErr := decodeCursor(&c)
			require.NotNil(t, apiErr, "malformed cursor should error")
			assert.Equal(t, apierror.ErrorCodeParameterInvalid, apiErr.Code)
		})
	}
}

// base64Raw mirrors encodeCursor's encoding for crafting malformed-but-decodable payloads.
func base64Raw(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

// ── Deterministic notification id ──────────────────────────────────

func TestDeterministicNotificationID(t *testing.T) {
	a := deterministicNotificationID("msg_1", "acus_a")
	b := deterministicNotificationID("msg_1", "acus_a")
	c := deterministicNotificationID("msg_1", "acus_b")
	d := deterministicNotificationID("msg_2", "acus_a")

	assert.Equal(t, a, b, "same (seed, recipient) must be stable for redelivery idempotency")
	assert.NotEqual(t, a, c, "different recipient must differ")
	assert.NotEqual(t, a, d, "different seed must differ")
	assert.True(t, strings.HasPrefix(a, "nf_"), "id should carry the nf_ prefix, got %q", a)
	assert.Len(t, a, len("nf_")+22, "id should be nf_ + 22 chars")
}

// ── SendNotification ───────────────────────────────────────────────

func TestSendNotification_PermissionDenied(t *testing.T) {
	svc := newSvc(t, nil, &fakeOutboxRepo{})
	ctx := identityCtx() // no alerts:create
	target := testRecipient

	_, apiErr := svc.SendNotification(ctx, domain.SendNotificationInput{
		Category:   "order.updated",
		Title:      "hi",
		TargetType: "account_user",
		TargetID:   target,
	})
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeInsufficientPerms, apiErr.Code)
}

func TestSendNotification_Validation(t *testing.T) {
	target := testRecipient
	ctx := identityCtx("alerts:create")

	t.Run("missing title", func(t *testing.T) {
		svc := newSvc(t, nil, &fakeOutboxRepo{})
		_, apiErr := svc.SendNotification(ctx, domain.SendNotificationInput{Category: "order.updated", TargetType: "account_user", TargetID: target})
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.ErrorCodeParameterMissing, apiErr.Code)
	})

	t.Run("missing target", func(t *testing.T) {
		svc := newSvc(t, nil, &fakeOutboxRepo{})
		_, apiErr := svc.SendNotification(ctx, domain.SendNotificationInput{Category: "order.updated", Title: "hi"})
		require.NotNil(t, apiErr)
		assert.Equal(t, apierror.ErrorCodeParameterMissing, apiErr.Code)
	})

	t.Run("broadcast creates an account-scoped announcement", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		factory := factorymock.NewMockRepoFactory(ctrl)
		announceRepo := repositorymock.NewMockAnnouncementRepo(ctrl)
		factory.EXPECT().NewAnnouncementRepo().Return(announceRepo).AnyTimes()
		factory.EXPECT().NewOutboxRepo().Return(&fakeOutboxRepo{}).AnyTimes()

		var captured *domain.CreateAnnouncementInput
		announceRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string, in *domain.CreateAnnouncementInput) *apierror.APIError {
				captured = in
				return nil
			})

		svc := NewMessagingSvc(factory)
		enqueued, apiErr := svc.SendNotification(ctx, domain.SendNotificationInput{Category: "system.broadcast", Title: "hi", TargetType: "account", TargetID: testAccountID})
		require.Nil(t, apiErr)
		assert.Equal(t, int64(1), enqueued)
		require.NotNil(t, captured)
		assert.Equal(t, "account", captured.Scope)
		require.NotNil(t, captured.AccountID)
		assert.Equal(t, testAccountID, *captured.AccountID)
	})
}

func TestMarkSeen_EmitsUnreadChanged(t *testing.T) {
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	outbox := &fakeOutboxRepo{}
	svc := newSvc(t, notifRepo, outbox)
	ctx := identityCtx()

	notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), testUserID, testAccountID).Return(testRecipient, nil)
	notifRepo.EXPECT().MarkSeen(gomock.Any(), "nf_1", testRecipient).Return(&domain.Notification{ID: "nf_1"}, nil)

	_, apiErr := svc.MarkSeen(ctx, "nf_1")
	require.Nil(t, apiErr)

	require.Len(t, outbox.inputs, 1, "marking emits one unread.changed realtime event")
	var rt messaging.RealtimeDeliveryData
	require.NoError(t, json.Unmarshal(outbox.inputs[0].Payload.Data, &rt))
	assert.Equal(t, "unread.changed", rt.Event)
	assert.Equal(t, testUserID, rt.RecipientUserID, "the unread hint targets the caller's user topic")
}

func TestSendNotification_BroadcastEmitsAnnouncementRealtime(t *testing.T) {
	ctrl := gomock.NewController(t)
	factory := factorymock.NewMockRepoFactory(ctrl)
	announceRepo := repositorymock.NewMockAnnouncementRepo(ctrl)
	outbox := &fakeOutboxRepo{}
	factory.EXPECT().NewAnnouncementRepo().Return(announceRepo).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()
	announceRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	svc := NewMessagingSvc(factory)
	_, apiErr := svc.SendNotification(identityCtx("alerts:create"), domain.SendNotificationInput{
		Category: "system.broadcast", Title: "hi", TargetType: "account", TargetID: testAccountID,
	})
	require.Nil(t, apiErr)

	require.Len(t, outbox.inputs, 1, "a broadcast emits one announcement.created realtime event")
	var rt messaging.RealtimeDeliveryData
	require.NoError(t, json.Unmarshal(outbox.inputs[0].Payload.Data, &rt))
	assert.Equal(t, "announcement.created", rt.Event)
	assert.Equal(t, testAccountID, rt.AccountID)
	assert.NotEmpty(t, rt.AnnouncementID, "the event carries the announcement id for the account topic")
}

func TestSendNotification_WritesFanoutOutbox(t *testing.T) {
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	// The sender is derived from the caller's identity: a user is attributed by their resolved account_user id.
	notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), testUserID, testAccountID).Return("acus_sender", nil)

	outbox := &fakeOutboxRepo{}
	svc := newSvc(t, notifRepo, outbox)
	ctx := identityCtx("alerts:create")
	target := testRecipient
	body := "the body"

	enqueued, apiErr := svc.SendNotification(ctx, domain.SendNotificationInput{
		Category:   "order.updated",
		Title:      "Order updated",
		Body:       &body,
		TargetType: "account_user",
		TargetID:   target,
	})
	require.Nil(t, apiErr)
	assert.Equal(t, int64(1), enqueued)

	require.Len(t, outbox.inputs, 1, "exactly one fan-out intent should be enqueued")
	in := outbox.inputs[0]
	assert.Equal(t, string(contracts.NotificationCmdFanout), in.MessageType)
	assert.Equal(t, string(contracts.NotificationCmdFanout), in.RoutingKey)
	assert.Equal(t, messaging.ApplicationExchange, in.Destination)

	var data messaging.AlertFanoutData
	require.NoError(t, json.Unmarshal(in.Payload.Data, &data))
	assert.Equal(t, testAccountID, data.AccountID)
	assert.Equal(t, "order.updated", data.Category)
	assert.Equal(t, "Order updated", data.Title)
	assert.Equal(t, "the body", data.Body)
	assert.Equal(t, []string{target}, data.RecipientAccountUserIDs)
	// Sender attribution comes from the authenticated user identity.
	assert.Equal(t, "user", data.SenderType)
	assert.Equal(t, "acus_sender", data.SenderID)
}

// ── FanOut ─────────────────────────────────────────────────────────

func TestFanOut_CreatesRowsAndRealtime(t *testing.T) {
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	outbox := &fakeOutboxRepo{}
	svc := newSvc(t, notifRepo, outbox)

	// FanOut resolves each account_user recipient back to its user id so the realtime push
	// can target the WS user-topic.
	notifRepo.EXPECT().ResolveUserID(gomock.Any(), "acus_a").Return("us_a", nil)
	notifRepo.EXPECT().ResolveUserID(gomock.Any(), "acus_b").Return("us_b", nil)

	var captured []*domain.Notification
	notifRepo.EXPECT().CreateBatch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ns []*domain.Notification) *apierror.APIError {
			captured = ns
			return nil
		})

	data := messaging.AlertFanoutData{
		AccountID:               testAccountID,
		Category:                "order.updated",
		Title:                   "Order updated",
		RecipientAccountUserIDs: []string{"acus_a", "acus_b"},
	}
	apiErr := svc.FanOut(context.Background(), "msg_seed", data)
	require.Nil(t, apiErr)

	require.Len(t, captured, 2, "one notification row per recipient")
	assert.Equal(t, "acus_a", captured[0].RecipientAccountUserID)
	assert.Equal(t, "acus_b", captured[1].RecipientAccountUserID)
	assert.Equal(t, "order.updated", captured[0].Category)
	assert.Equal(t, deterministicNotificationID("msg_seed", "acus_a"), captured[0].ID,
		"row id must be the deterministic id (redelivery idempotency)")

	// One realtime delivery per recipient.
	require.Len(t, outbox.inputs, 2)
	for _, in := range outbox.inputs {
		assert.Equal(t, string(contracts.NotificationEventDelivered), in.RoutingKey)
	}
}

func TestFanOut_ResolvesRecipientUserIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	outbox := &fakeOutboxRepo{}
	svc := newSvc(t, notifRepo, outbox)

	// Producers that hold a user id (e.g. agent-service) supply RecipientUserIDs; FanOut
	// resolves them to the per-account account_user id the row is keyed by.
	notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), "us_x", testAccountID).Return("acus_x", nil)

	var captured []*domain.Notification
	notifRepo.EXPECT().CreateBatch(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, ns []*domain.Notification) *apierror.APIError {
			captured = ns
			return nil
		})

	data := messaging.AlertFanoutData{
		AccountID:        testAccountID,
		Category:         "agent.run_completed",
		Title:            "Run finished",
		SenderType:       "agent",
		SenderID:         "agtc_1",
		SenderName:       "Forecaster",
		RecipientUserIDs: []string{"us_x"},
	}
	require.Nil(t, svc.FanOut(context.Background(), "run_seed", data))

	require.Len(t, captured, 1)
	assert.Equal(t, "acus_x", captured[0].RecipientAccountUserID)
	require.NotNil(t, captured[0].SenderType)
	assert.Equal(t, "agent", *captured[0].SenderType)
	require.Len(t, outbox.inputs, 1)
}

func TestFanOut_BroadcastAndEmptyAreNoOps(t *testing.T) {
	// No repo calls expected (mock with no EXPECT fails on any call), so passing nil-ish
	// repos via a controller with strict expectations proves no writes happen.
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	outbox := &fakeOutboxRepo{}
	svc := newSvc(t, notifRepo, outbox)

	require.Nil(t, svc.FanOut(context.Background(), "seed", messaging.AlertFanoutData{Broadcast: true}))
	require.Nil(t, svc.FanOut(context.Background(), "seed", messaging.AlertFanoutData{RecipientAccountUserIDs: nil}))
	assert.Empty(t, outbox.inputs, "broadcast/empty fan-out must not write realtime rows")
}

// ── ListNotifications: limit clamp + page probe ────────────────────

func TestListNotifications_LimitClampAndNextPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	svc := newSvc(t, notifRepo, nil)
	ctx := identityCtx()

	notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), testUserID, testAccountID).Return(testRecipient, nil).AnyTimes()

	now := time.Now().UTC()
	// Default limit (0 → 50): repo is queried with limit+1 = 51, and a full page+1 of rows
	// yields HasNextPage + a NextCursor, truncated back to 50.
	var gotFilter domain.NotificationListFilter
	rows := make([]*domain.Notification, 51)
	for i := range rows {
		rows[i] = &domain.Notification{ID: "nf_x", RecipientAccountUserID: testRecipient, CreatedAt: now}
	}
	notifRepo.EXPECT().List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f domain.NotificationListFilter) ([]*domain.Notification, *apierror.APIError) {
			gotFilter = f
			return rows, nil
		})

	page, apiErr := svc.ListNotifications(ctx, domain.ListNotificationsInput{Limit: 0})
	require.Nil(t, apiErr)
	assert.Equal(t, int32(51), gotFilter.Limit, "default limit 50 → query limit+1")
	assert.Len(t, page.Notifications, 50, "page truncated to the requested limit")
	assert.True(t, page.HasNextPage)
	require.NotNil(t, page.NextCursor)
}

func TestListNotifications_LimitMaxClamp(t *testing.T) {
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	svc := newSvc(t, notifRepo, nil)
	ctx := identityCtx()

	notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), testUserID, testAccountID).Return(testRecipient, nil)
	var gotFilter domain.NotificationListFilter
	notifRepo.EXPECT().List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f domain.NotificationListFilter) ([]*domain.Notification, *apierror.APIError) {
			gotFilter = f
			return nil, nil
		})

	_, apiErr := svc.ListNotifications(ctx, domain.ListNotificationsInput{Limit: 5000})
	require.Nil(t, apiErr)
	assert.Equal(t, int32(101), gotFilter.Limit, "limit clamped to max 100 → query limit+1")
}

// ── recipient(): graceful empty for actors without an account_user ─

func TestList_GracefulEmptyWhenNoAccountUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	svc := newSvc(t, notifRepo, nil)
	ctx := identityCtx()

	// No account_user (e.g. an API key) → ResolveAccountUserID returns not-found.
	notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), testUserID, testAccountID).
		Return("", apierror.NewResourceNotFoundError("not found"))
	// List is still called, but with an empty recipient (yields an empty feed).
	notifRepo.EXPECT().List(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, f domain.NotificationListFilter) ([]*domain.Notification, *apierror.APIError) {
			assert.Empty(t, f.RecipientAccountUserID, "no-account-user actor resolves to an empty recipient")
			return nil, nil
		})

	page, apiErr := svc.ListNotifications(ctx, domain.ListNotificationsInput{})
	require.Nil(t, apiErr, "a list for an actor without a feed should not error")
	assert.Empty(t, page.Notifications)
}

func TestRecipient_PropagatesNonNotFoundError(t *testing.T) {
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	svc := newSvc(t, notifRepo, nil)
	ctx := identityCtx()

	notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), testUserID, testAccountID).
		Return("", apierror.NewInternalError(nil, "db down"))

	_, apiErr := svc.GetUnreadCount(ctx)
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeInternalError, apiErr.Code)
}

func TestActor_RequiresAuthentication(t *testing.T) {
	svc := newSvc(t, nil, nil)
	_, apiErr := svc.GetUnreadCount(context.Background()) // no identity
	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeInvalidCredentials, apiErr.Code)
}

// ── Announcements ──────────────────────────────────────────────────

// newSvcWithRepos wires the messaging service with a notification and announcement repo, plus
// a fake outbox so best-effort realtime emits (unread.changed / announcement.created) succeed.
func newSvcWithRepos(t *testing.T, notifRepo domain.NotificationRepo, announceRepo domain.AnnouncementRepo) domain.MessagingSvc {
	t.Helper()
	ctrl := gomock.NewController(t)
	factory := factorymock.NewMockRepoFactory(ctrl)
	if notifRepo != nil {
		factory.EXPECT().NewNotificationRepo().Return(notifRepo).AnyTimes()
	}
	if announceRepo != nil {
		factory.EXPECT().NewAnnouncementRepo().Return(announceRepo).AnyTimes()
	}
	factory.EXPECT().NewOutboxRepo().Return(&fakeOutboxRepo{}).AnyTimes()
	return NewMessagingSvc(factory)
}

func TestGetUnreadCount_IncludesAnnouncements(t *testing.T) {
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	announceRepo := repositorymock.NewMockAnnouncementRepo(ctrl)
	svc := newSvcWithRepos(t, notifRepo, announceRepo)
	ctx := identityCtx()

	notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), testUserID, testAccountID).Return(testRecipient, nil)
	notifRepo.EXPECT().CountUnseen(gomock.Any(), testRecipient).Return(int64(2), nil)
	announceRepo.EXPECT().CountUnseen(gomock.Any(), testRecipient, gomock.Any()).Return(int64(3), nil)

	counts, apiErr := svc.GetUnreadCount(ctx)
	require.Nil(t, apiErr)
	assert.Equal(t, int64(2), counts.Notifications, "per-user sub-count excludes announcements")
	assert.Equal(t, int64(5), counts.Total, "total folds in unseen announcements")
}

func TestGetUnreadSummary_MergesAccounts(t *testing.T) {
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	announceRepo := repositorymock.NewMockAnnouncementRepo(ctrl)
	svc := newSvcWithRepos(t, notifRepo, announceRepo)
	ctx := identityCtx()

	notifRepo.EXPECT().CountUnseenByUserAccounts(gomock.Any(), testUserID).Return([]domain.AccountUnread{
		{AccountID: "ac_a", Unread: 2},
		{AccountID: "ac_b", Unread: 1},
	}, nil)
	announceRepo.EXPECT().CountUnseenByUserAccounts(gomock.Any(), testUserID).Return([]domain.AccountUnread{
		{AccountID: "ac_a", Unread: 3},
	}, nil)

	summary, apiErr := svc.GetUnreadSummary(ctx)
	require.Nil(t, apiErr)
	assert.Equal(t, int64(6), summary.Total)

	byAccount := map[string]int64{}
	for _, a := range summary.Accounts {
		byAccount[a.AccountID] = a.Unread
	}
	assert.Equal(t, int64(5), byAccount["ac_a"], "notification + announcement counts merge per account")
	assert.Equal(t, int64(1), byAccount["ac_b"])
}

func TestListAnnouncements_LimitClampAndNextPage(t *testing.T) {
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	announceRepo := repositorymock.NewMockAnnouncementRepo(ctrl)
	svc := newSvcWithRepos(t, notifRepo, announceRepo)
	ctx := identityCtx()

	notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), testUserID, testAccountID).Return(testRecipient, nil)

	// Return limit+1 rows so the service trims and reports a next page.
	announceRepo.EXPECT().ListActive(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, f domain.AnnouncementListFilter) ([]*domain.Announcement, *apierror.APIError) {
			assert.Equal(t, int32(3), f.Limit, "fetches limit+1 to probe for a next page")
			rows := make([]*domain.Announcement, 3)
			for i := range rows {
				rows[i] = &domain.Announcement{ID: "an_" + string(rune('a'+i)), PublishAt: time.Unix(int64(100-i), 0)}
			}
			return rows, nil
		})

	page, apiErr := svc.ListAnnouncements(ctx, domain.ListAnnouncementsInput{Limit: 2})
	require.Nil(t, apiErr)
	assert.Len(t, page.Announcements, 2)
	assert.True(t, page.HasNextPage)
	require.NotNil(t, page.NextCursor)
}

func TestMarkAnnouncementSeen_VisibilityCheckedThenReturnsUpdated(t *testing.T) {
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	announceRepo := repositorymock.NewMockAnnouncementRepo(ctrl)
	svc := newSvcWithRepos(t, notifRepo, announceRepo)
	ctx := identityCtx()

	notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), testUserID, testAccountID).Return(testRecipient, nil)
	// Visibility check, then mark, then re-fetch (two GetActiveByID calls).
	announceRepo.EXPECT().GetActiveByID(gomock.Any(), "an_1", testRecipient, gomock.Any()).
		Return(&domain.Announcement{ID: "an_1"}, nil).Times(2)
	announceRepo.EXPECT().MarkSeen(gomock.Any(), "an_1", testRecipient).Return(nil)

	a, apiErr := svc.MarkAnnouncementSeen(ctx, "an_1")
	require.Nil(t, apiErr)
	assert.Equal(t, "an_1", a.ID)
}
