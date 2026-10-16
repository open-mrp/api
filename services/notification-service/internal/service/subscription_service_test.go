package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/messaging"
	"github.com/open-mrp/api/shared/notifyprefs"
	apierror "github.com/open-mrp/apikit/apierror"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type fanoutFixture struct {
	svc       domain.MessagingSvc
	notifRepo *repositorymock.MockNotificationRepo
	subRepo   *repositorymock.MockSubscriptionRepo
	prefRepo  *repositorymock.MockNotificationPreferenceRepo
	outbox    *fakeOutboxRepo
	created   []*domain.Notification
}

func newFanoutFixture(t *testing.T, account notifyprefs.Defaults) *fanoutFixture {
	ctrl := gomock.NewController(t)
	fx := &fanoutFixture{
		notifRepo: repositorymock.NewMockNotificationRepo(ctrl),
		subRepo:   repositorymock.NewMockSubscriptionRepo(ctrl),
		prefRepo:  repositorymock.NewMockNotificationPreferenceRepo(ctrl),
		outbox:    &fakeOutboxRepo{},
	}
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewNotificationRepo().Return(fx.notifRepo).AnyTimes()
	factory.EXPECT().NewSubscriptionRepo().Return(fx.subRepo).AnyTimes()
	factory.EXPECT().NewNotificationPreferenceRepo().Return(fx.prefRepo).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(fx.outbox).AnyTimes()
	stubNotificationSettings(ctrl, factory, account, notifyprefs.Empty())
	fx.svc = NewMessagingSvc(factory, passthroughTx{f: factory})

	fx.notifRepo.EXPECT().ResolveUserID(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, acus string) (string, *apierror.APIError) { return "us_" + acus, nil }).AnyTimes()
	fx.notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, userID, _ string) (string, *apierror.APIError) {
			return userID[len("us_"):], nil
		}).AnyTimes()
	fx.notifRepo.EXPECT().CreateBatch(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, ns []*domain.Notification) *apierror.APIError {
			fx.created = append(fx.created, ns...)
			return nil
		}).AnyTimes()
	return fx
}

func recipientsOf(ns []*domain.Notification) []string {
	out := make([]string, 0, len(ns))
	for _, n := range ns {
		out = append(out, n.RecipientAccountUserID)
	}
	return out
}

var order = messaging.RecordRef{Type: string(constants.RecordTypeSalesOrder), ID: "so_1"}
var customer = messaging.RecordRef{Type: string(constants.RecordTypeCustomer), ID: "ac_cust"}

// Subscribers of the order and of its customer are told, except the person who made the change.
func TestFanOut_AudienceIsSubscribersMinusActor(t *testing.T) {
	fx := newFanoutFixture(t, notifyprefs.Empty())
	fx.prefRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	fx.subRepo.EXPECT().Subscribers(gomock.Any(), testAccountID, order.Type, order.ID).Return([]string{"acus_rep", "acus_actor"}, nil)
	fx.subRepo.EXPECT().Subscribers(gomock.Any(), testAccountID, customer.Type, customer.ID).Return([]string{"acus_cs", "acus_rep"}, nil)

	apiErr := fx.svc.FanOut(context.Background(), "seed", messaging.AlertFanoutData{
		AccountID: testAccountID, Category: string(constants.NotificationCategoryOrderUpdated), Title: "Sales order 7 updated",
		Audience: &messaging.SubscriberAudience{Records: []messaging.RecordRef{order, customer}, ExcludeUserIDs: []string{"us_acus_actor"}},
	})
	require.Nil(t, apiErr)
	assert.ElementsMatch(t, []string{"acus_rep", "acus_cs"}, recipientsOf(fx.created))
}

// An order nobody has ever subscribed to (made before subscriptions) falls back to its owner.
func TestFanOut_AudienceFallsBackWhenRecordHasNoRows(t *testing.T) {
	fx := newFanoutFixture(t, notifyprefs.Empty())
	fx.prefRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	fx.subRepo.EXPECT().Subscribers(gomock.Any(), testAccountID, order.Type, order.ID).Return(nil, nil)
	fx.subRepo.EXPECT().HasAny(gomock.Any(), testAccountID, order.Type, order.ID).Return(false, nil)

	apiErr := fx.svc.FanOut(context.Background(), "seed", messaging.AlertFanoutData{
		AccountID: testAccountID, Category: string(constants.NotificationCategoryOrderUpdated), Title: "t",
		Audience: &messaging.SubscriberAudience{Records: []messaging.RecordRef{order}, FallbackUserIDs: []string{"us_acus_owner"}},
	})
	require.Nil(t, apiErr)
	assert.Equal(t, []string{"acus_owner"}, recipientsOf(fx.created))
}

// Everyone unsubscribed: nobody is told, and the owner fallback doesn't apply.
func TestFanOut_AudienceAllUnsubscribedTellsNobody(t *testing.T) {
	fx := newFanoutFixture(t, notifyprefs.Empty())
	fx.subRepo.EXPECT().Subscribers(gomock.Any(), testAccountID, order.Type, order.ID).Return(nil, nil)
	fx.subRepo.EXPECT().HasAny(gomock.Any(), testAccountID, order.Type, order.ID).Return(true, nil)

	apiErr := fx.svc.FanOut(context.Background(), "seed", messaging.AlertFanoutData{
		AccountID: testAccountID, Category: string(constants.NotificationCategoryOrderUpdated), Title: "t",
		Audience: &messaging.SubscriberAudience{Records: []messaging.RecordRef{order}, FallbackUserIDs: []string{"us_acus_owner"}},
	})
	require.Nil(t, apiErr)
	assert.Empty(t, fx.created)
}

// Subscribe intents insert a row only for people whose auto-subscribe setting includes the reason.
func TestFanOut_SubscribeIntentsFollowAutoSubscribe(t *testing.T) {
	fx := newFanoutFixture(t, notifyprefs.Empty())
	fx.prefRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
	var inserted []*domain.Subscription
	fx.subRepo.EXPECT().Insert(gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, s *domain.Subscription) (bool, *apierror.APIError) {
			inserted = append(inserted, s)
			return true, nil
		}).AnyTimes()

	apiErr := fx.svc.FanOut(context.Background(), "seed", messaging.AlertFanoutData{
		AccountID: testAccountID,
		Record:    &order,
		Subscribe: []messaging.SubscriptionIntent{
			{UserID: "us_acus_creator", Reason: string(constants.SubscriptionReasonCreator)},
			{AccountUserID: "acus_rep", Reason: string(constants.SubscriptionReasonAssignee)},
			{UserID: "us_acus_editor", Reason: string(constants.SubscriptionReasonEditor)},
		},
	})
	require.Nil(t, apiErr)
	require.Len(t, inserted, 2, "editing doesn't subscribe by default")
	assert.Equal(t, "acus_creator", inserted[0].AccountUserID)
	assert.Equal(t, string(constants.SubscriptionReasonCreator), inserted[0].Reason)
	assert.Equal(t, "acus_rep", inserted[1].AccountUserID)
	assert.Equal(t, order.ID, inserted[1].RecordID)
	assert.Empty(t, fx.created, "a subscribe-only payload delivers nothing")
}

// A type whose resolved channels exclude in_app writes no bell row; email on instant enqueues a notification email.
func TestFanOut_ChannelsGateBellAndEmail(t *testing.T) {
	orderType := string(constants.NotificationCategoryOrderUpdated)
	account := notifyprefs.Defaults{Types: map[string]notifyprefs.Setting{orderType: {
		Channels: []constants.NotificationChannel{constants.NotificationChannelEmail}, Digest: constants.NotificationDigestInstant,
	}}}
	fx := newFanoutFixture(t, account)
	fx.prefRepo.EXPECT().List(gomock.Any(), "acus_a").Return(nil, nil).AnyTimes()
	fx.prefRepo.EXPECT().List(gomock.Any(), "acus_b").Return([]*domain.NotificationPreference{
		{Category: orderType, InAppEnabled: true, Digest: "instant"},
	}, nil).AnyTimes()
	fx.notifRepo.EXPECT().ResolveRecipientContact(gomock.Any(), "acus_a").Return(&domain.RecipientContact{Email: "a@example.com"}, nil)

	apiErr := fx.svc.FanOut(context.Background(), "seed", messaging.AlertFanoutData{
		AccountID: testAccountID, Category: orderType, Title: "t", RecipientAccountUserIDs: []string{"acus_a", "acus_b"},
	})
	require.Nil(t, apiErr)
	assert.Equal(t, []string{"acus_b"}, recipientsOf(fx.created), "only b keeps in_app")
	emails := 0
	for _, in := range fx.outbox.inputs {
		if in.RoutingKey == "notification.cmd.send_email" {
			emails++
		}
	}
	assert.Equal(t, 1, emails, "a is emailed")
}
