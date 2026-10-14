package service

import (
	"context"
	"testing"
	"time"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/appctx"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	"github.com/open-mrp/apikit/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// passthroughTx runs the transaction body against the same mocked factory.
type passthroughTx struct{ f domain.RepoFactory }

func (p passthroughTx) WithTx(ctx context.Context, fn func(context.Context, domain.RepoFactory) *apierror.APIError) *apierror.APIError {
	return fn(ctx, p.f)
}

func (p passthroughTx) WithTxSavepoint(ctx context.Context, fn func(context.Context, domain.RepoFactory, db.SavepointRunner) *apierror.APIError) *apierror.APIError {
	return apierror.NewInvariantViolationError("savepoints are not used here")
}

type rescheduleHarness struct {
	svc          domain.ConversationSvc
	messages     *repositorymock.MockMessageRepo
	participants *repositorymock.MockParticipantRepo
}

const reschedulerAcus = "acus_scheduler"

func newRescheduleHarness(t *testing.T) *rescheduleHarness {
	t.Helper()
	ctrl := gomock.NewController(t)
	h := &rescheduleHarness{
		messages:     repositorymock.NewMockMessageRepo(ctrl),
		participants: repositorymock.NewMockParticipantRepo(ctrl),
	}
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), testUserID, testAccountID).Return(reschedulerAcus, nil).AnyTimes()
	idemRepo := repositorymock.NewMockIdempotencyKeyRepo(ctrl)
	idemRepo.EXPECT().GetByScopeHash(gomock.Any(), gomock.Any()).Return(&domain.IdempotencyKey{TypeID: "idk_resched", RecoveryPoint: string(domain.RecoveryPointStarted)}, nil).AnyTimes()
	idemRepo.EXPECT().SetResponse(gomock.Any(), "idk_resched", gomock.Any(), gomock.Any(), domain.RecoveryPointFinished).Return(nil).AnyTimes()

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewNotificationRepo().Return(notifRepo).AnyTimes()
	factory.EXPECT().NewIdempotencyKeyRepo().Return(idemRepo).AnyTimes()
	factory.EXPECT().NewMessageRepo().Return(h.messages).AnyTimes()
	factory.EXPECT().NewParticipantRepo().Return(h.participants).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(&fakeOutboxRepo{}).AnyTimes()
	h.svc = NewConversationSvc(factory, passthroughTx{f: factory}, nil, "", nil, nil, "")
	return h
}

func rescheduleCtx() context.Context {
	return appctx.WithHandler(appctx.WithIdempotencyKey(identityCtx("messaging:update"), "idem-resched"), "/notification.ChatService/RescheduleMessage")
}

func TestRescheduleMessage_MovesTheCallersScheduledMessage(t *testing.T) {
	t.Parallel()
	h := newRescheduleHarness(t)
	at := time.Now().Add(2 * time.Hour)
	body := "moved to the afternoon"
	moved := &domain.Message{ID: "mg_sched", Status: string(constants.MessageStatusScheduled), Body: &body, ScheduledFor: &at}
	was := time.Now().Add(time.Hour)
	original := &domain.Message{ID: "mg_sched", Status: string(constants.MessageStatusScheduled), ScheduledFor: &was}

	gomock.InOrder(
		h.messages.EXPECT().GetByID(gomock.Any(), "mg_sched", testAccountID).Return(original, nil),
		h.messages.EXPECT().Reschedule(gomock.Any(), "mg_sched", testAccountID, reschedulerAcus, at, &body, gomock.Any()).
			DoAndReturn(func(_ context.Context, _, _, _ string, _ time.Time, _, preview *string) (bool, *apierror.APIError) {
				require.NotNil(t, preview)
				assert.Equal(t, body, *preview, "the preview follows the new body")
				return true, nil
			}),
		h.messages.EXPECT().GetByID(gomock.Any(), "mg_sched", testAccountID).Return(moved, nil),
	)

	got, apiErr := h.svc.RescheduleMessage(rescheduleCtx(), domain.RescheduleMessageInput{ID: "mg_sched", ScheduledFor: at, Body: &body})

	require.Nil(t, apiErr)
	assert.Same(t, moved, got)
}

func TestRescheduleMessage_RefusesWhatItCannotMove(t *testing.T) {
	t.Parallel()
	senderPID := "cvpt_sender"
	other := "acus_someone_else"
	mine := reschedulerAcus

	tests := []struct {
		name     string
		message  *domain.Message
		getErr   *apierror.APIError
		sender   *string
		wantCode apierror.Code
	}{
		{"a message that does not exist", nil, apierror.NewNotFoundError("not found"), nil, apierror.CodeResourceNotFound},
		{"someone else's scheduled message", &domain.Message{ID: "mg_x", ConversationID: "cv_x", Status: string(constants.MessageStatusScheduled), SenderParticipantID: &senderPID}, nil, &other, apierror.CodeResourceNotFound},
		{"a message already sent", &domain.Message{ID: "mg_x", ConversationID: "cv_x", Status: string(constants.MessageStatusSent), SenderParticipantID: &senderPID}, nil, &mine, apierror.CodeValidationFailed},
		{"a canceled message", &domain.Message{ID: "mg_x", ConversationID: "cv_x", Status: string(constants.MessageStatusCanceled), SenderParticipantID: &senderPID}, nil, &mine, apierror.CodeValidationFailed},
		{"a message whose time has come", &domain.Message{ID: "mg_x", ConversationID: "cv_x", Status: string(constants.MessageStatusScheduled), SenderParticipantID: &senderPID}, nil, &mine, apierror.CodeValidationFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			h := newRescheduleHarness(t)
			h.messages.EXPECT().Reschedule(gomock.Any(), "mg_x", testAccountID, reschedulerAcus, gomock.Any(), nil, nil).Return(false, nil)
			// Read once ahead of the move for the audit pre-image, and again to explain the refusal.
			h.messages.EXPECT().GetByID(gomock.Any(), "mg_x", testAccountID).Return(tt.message, tt.getErr).Times(2)
			if tt.sender != nil {
				h.participants.EXPECT().GetByID(gomock.Any(), senderPID, "cv_x").Return(&domain.ConversationParticipant{ID: senderPID, AccountUserID: tt.sender}, nil)
			}

			_, apiErr := h.svc.RescheduleMessage(rescheduleCtx(), domain.RescheduleMessageInput{ID: "mg_x", ScheduledFor: time.Now().Add(time.Hour)})

			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantCode, apiErr.Code)
		})
	}
}

func TestRescheduleMessage_ValidatesBeforeTouchingTheMessage(t *testing.T) {
	t.Parallel()
	blank := ""

	for name, input := range map[string]domain.RescheduleMessageInput{
		"a time that has passed": {ID: "mg_x", ScheduledFor: time.Now().Add(-time.Minute)},
		"a blank body":           {ID: "mg_x", ScheduledFor: time.Now().Add(time.Hour), Body: &blank},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			h := newRescheduleHarness(t)

			_, apiErr := h.svc.RescheduleMessage(rescheduleCtx(), input)

			require.NotNil(t, apiErr)
			assert.Equal(t, apierror.CodeParameterInvalid, apiErr.Code)
		})
	}
}
