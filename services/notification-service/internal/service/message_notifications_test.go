package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func mnPtr[T any](v T) *T { return &v }

// fanoutMessageNotifications must skip muted recipients entirely (no bell row, no email) and skip
// the sender, while creating exactly one bell row for an active recipient whose in-app channel is
// on and email channel is off.
func TestFanoutMessageNotifications_MuteAndPreferenceGating(t *testing.T) {
	ctrl := gomock.NewController(t)

	const (
		convID    = "cv_1"
		accountID = "ac_1"
		senderAc  = "acus_sender"
		mutedAc   = "acus_muted"
		activeAc  = "acus_active"
	)

	participantRepo := repositorymock.NewMockParticipantRepo(ctrl)
	participantRepo.EXPECT().List(gomock.Any(), convID).Return([]*domain.ConversationParticipant{
		{ID: "p_sender", AccountUserID: mnPtr(senderAc), Membership: string(constants.ParticipantMembershipActive)},
		{ID: "p_muted", AccountUserID: mnPtr(mutedAc), Membership: string(constants.ParticipantMembershipActive), Notifications: string(constants.ParticipantNotificationsMuted)},
		{ID: "p_active", AccountUserID: mnPtr(activeAc), Membership: string(constants.ParticipantMembershipActive)},
	}, nil)

	convRepo := repositorymock.NewMockConversationRepo(ctrl)
	convRepo.EXPECT().GetByID(gomock.Any(), convID, accountID).Return(&domain.Conversation{ID: convID}, nil)

	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	// Sender display name resolution for attribution.
	notifRepo.EXPECT().ResolveRecipientContact(gomock.Any(), senderAc).Return(&domain.RecipientContact{Name: "Dane"}, nil)
	// Exactly one bell row, for the active (non-muted) recipient.
	notifRepo.EXPECT().Create(gomock.Any(), gomock.Cond(func(n any) bool {
		notif, ok := n.(*domain.Notification)
		return ok && notif.RecipientAccountUserID == activeAc && notif.Category == string(constants.NotificationCategoryChatMessage)
	})).Return(nil)

	prefRepo := repositorymock.NewMockNotificationPreferenceRepo(ctrl)
	// Active recipient: in-app on, email off → bell row, no email.
	prefRepo.EXPECT().GetEffective(gomock.Any(), activeAc, string(constants.NotificationCategoryChatMessage)).
		Return(&domain.EffectiveNotificationPreference{InAppEnabled: true, EmailEnabled: false, Digest: "instant"}, nil)

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewParticipantRepo().Return(participantRepo).AnyTimes()
	factory.EXPECT().NewConversationRepo().Return(convRepo).AnyTimes()
	factory.EXPECT().NewNotificationRepo().Return(notifRepo).AnyTimes()
	factory.EXPECT().NewNotificationPreferenceRepo().Return(prefRepo).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(&fakeOutboxRepo{}).AnyTimes()

	svc := &conversationSvcImpl{repoFactory: factory}
	msg := &domain.Message{ID: "mg_1", ConversationID: convID, AccountID: accountID, Preview: mnPtr("hi there")}

	apiErr := svc.fanoutMessageNotifications(context.Background(), factory, msg, accountID, senderAc)
	require.Nil(t, apiErr)
}

// An explicit email-on/instant preference drives the email bridge; the bell sender attribution
// resolves to the authoring account user's name. (With no preference row, email defaults off — covered separately.)
func TestFanoutMessageNotifications_DefaultsAndEmailBridge(t *testing.T) {
	ctrl := gomock.NewController(t)

	const (
		convID    = "cv_2"
		accountID = "ac_1"
		senderAc  = "acus_sender"
		activeAc  = "acus_active"
	)

	participantRepo := repositorymock.NewMockParticipantRepo(ctrl)
	participantRepo.EXPECT().List(gomock.Any(), convID).Return([]*domain.ConversationParticipant{
		{ID: "p_active", AccountUserID: mnPtr(activeAc), Membership: string(constants.ParticipantMembershipActive)},
	}, nil)

	convRepo := repositorymock.NewMockConversationRepo(ctrl)
	convRepo.EXPECT().GetByID(gomock.Any(), convID, accountID).Return(&domain.Conversation{ID: convID}, nil)

	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	// The bell sender name resolves to the authoring account user's contact name.
	notifRepo.EXPECT().ResolveRecipientContact(gomock.Any(), senderAc).Return(&domain.RecipientContact{Name: "Sender Bob"}, nil)
	notifRepo.EXPECT().Create(gomock.Any(), gomock.Cond(func(n any) bool {
		notif, ok := n.(*domain.Notification)
		return ok && notif.SenderName != nil && *notif.SenderName == "Sender Bob"
	})).Return(nil)
	// Email bridge resolves the recipient's email.
	notifRepo.EXPECT().ResolveRecipientContact(gomock.Any(), activeAc).Return(&domain.RecipientContact{Email: "x@y.com", Name: "Ada"}, nil)

	prefRepo := repositorymock.NewMockNotificationPreferenceRepo(ctrl)
	// Explicit opt-in: in-app on, email on, instant digest → bell row + email command.
	prefRepo.EXPECT().GetEffective(gomock.Any(), activeAc, string(constants.NotificationCategoryChatMessage)).
		Return(&domain.EffectiveNotificationPreference{InAppEnabled: true, EmailEnabled: true, Digest: "instant"}, nil)

	outbox := &fakeOutboxRepo{}

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewParticipantRepo().Return(participantRepo).AnyTimes()
	factory.EXPECT().NewConversationRepo().Return(convRepo).AnyTimes()
	factory.EXPECT().NewNotificationRepo().Return(notifRepo).AnyTimes()
	factory.EXPECT().NewNotificationPreferenceRepo().Return(prefRepo).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()

	svc := &conversationSvcImpl{repoFactory: factory}
	msg := &domain.Message{ID: "mg_2", ConversationID: convID, AccountID: accountID, Preview: mnPtr("hello")}

	apiErr := svc.fanoutMessageNotifications(context.Background(), factory, msg, accountID, senderAc)
	require.Nil(t, apiErr)
	require.Equal(t, 1, len(outbox.inputs), "an email command was enqueued")
}

// An agent-authored chat message titles the bell notification with the agent's display name.
func TestFanoutMessageNotifications_AgentSenderTitle(t *testing.T) {
	ctrl := gomock.NewController(t)

	const (
		convID    = "cv_agent"
		accountID = "ac_1"
		activeAc  = "acus_active"
		agentID   = "agdf_forecaster"
	)

	participantRepo := repositorymock.NewMockParticipantRepo(ctrl)
	participantRepo.EXPECT().List(gomock.Any(), convID).Return([]*domain.ConversationParticipant{
		{ID: "p_active", AccountUserID: mnPtr(activeAc), Membership: string(constants.ParticipantMembershipActive)},
	}, nil)

	convRepo := repositorymock.NewMockConversationRepo(ctrl)
	convRepo.EXPECT().GetByID(gomock.Any(), convID, accountID).Return(&domain.Conversation{ID: convID}, nil)

	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	notifRepo.EXPECT().Create(gomock.Any(), gomock.Cond(func(n any) bool {
		notif, ok := n.(*domain.Notification)
		return ok &&
			notif.RecipientAccountUserID == activeAc &&
			notif.Title == "Forecaster" &&
			notif.SenderName != nil && *notif.SenderName == "Forecaster"
	})).Return(nil)

	prefRepo := repositorymock.NewMockNotificationPreferenceRepo(ctrl)
	prefRepo.EXPECT().GetEffective(gomock.Any(), activeAc, string(constants.NotificationCategoryChatMessage)).
		Return(&domain.EffectiveNotificationPreference{InAppEnabled: true, EmailEnabled: false, Digest: "instant"}, nil)

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewParticipantRepo().Return(participantRepo).AnyTimes()
	factory.EXPECT().NewConversationRepo().Return(convRepo).AnyTimes()
	factory.EXPECT().NewNotificationRepo().Return(notifRepo).AnyTimes()
	factory.EXPECT().NewNotificationPreferenceRepo().Return(prefRepo).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(&fakeOutboxRepo{}).AnyTimes()

	svc := &conversationSvcImpl{repoFactory: factory}
	agentName := "Forecaster"
	msg := &domain.Message{
		ID:                  "mg_agent",
		ConversationID:      convID,
		AccountID:           accountID,
		Preview:             mnPtr("Here's your forecast."),
		SenderAgentConfigID: mnPtr(agentID),
		SenderAgentName:     &agentName,
	}

	apiErr := svc.fanoutMessageNotifications(context.Background(), factory, msg, accountID, "")
	require.Nil(t, apiErr)
}
