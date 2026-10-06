package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRequestedParticipantRoles_ValidatesEachEntryAsSetRoleDoes(t *testing.T) {
	t.Parallel()
	const owner = "acus_owner"

	tests := []struct {
		name     string
		entries  []domain.ParticipantRoleInput
		unknown  string
		wantCode apierror.ErrorCode
		param    string
	}{
		{"no account user", []domain.ParticipantRoleInput{{Role: "admin"}}, "", apierror.ErrorCodeParameterMissing, "participants[0].account_user_id"},
		{"the creator", []domain.ParticipantRoleInput{{AccountUserID: owner, Role: "admin"}}, "", apierror.ErrorCodeParameterInvalid, "participants[0].account_user_id"},
		{"a user listed twice", []domain.ParticipantRoleInput{{AccountUserID: "acus_a", Role: "admin"}, {AccountUserID: "acus_a", Role: "viewer"}}, "", apierror.ErrorCodeParameterInvalid, "participants[1].account_user_id"},
		{"an unknown role", []domain.ParticipantRoleInput{{AccountUserID: "acus_a", Role: "boss"}}, "", apierror.ErrorCodeParameterInvalid, "participants[0].role"},
		{"a user that does not exist", []domain.ParticipantRoleInput{{AccountUserID: "acus_gone", Role: "admin"}}, "acus_gone", apierror.ErrorCodeParameterInvalid, "participants[0].account_user_id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
			notifRepo.EXPECT().ResolveUserID(gomock.Any(), gomock.Any()).DoAndReturn(func(_ context.Context, acus string) (string, *apierror.APIError) {
				if acus == tt.unknown {
					return "", apierror.NewResourceNotFoundError("not found")
				}
				return "us_" + acus, nil
			}).AnyTimes()
			factory := factorymock.NewMockRepoFactory(ctrl)
			factory.EXPECT().NewNotificationRepo().Return(notifRepo).AnyTimes()
			svc := &conversationSvcImpl{repoFactory: factory}

			_, apiErr := svc.requestedParticipantRoles(context.Background(), tt.entries, owner)

			require.NotNil(t, apiErr)
			assert.Equal(t, tt.wantCode, apiErr.Code)
			assert.Equal(t, tt.param, apiErr.Param)
		})
	}

	t.Run("every valid role, owner included", func(t *testing.T) {
		t.Parallel()
		ctrl := gomock.NewController(t)
		notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
		notifRepo.EXPECT().ResolveUserID(gomock.Any(), gomock.Any()).Return("us_x", nil).Times(4)
		factory := factorymock.NewMockRepoFactory(ctrl)
		factory.EXPECT().NewNotificationRepo().Return(notifRepo).AnyTimes()
		svc := &conversationSvcImpl{repoFactory: factory}

		roles, apiErr := svc.requestedParticipantRoles(context.Background(), []domain.ParticipantRoleInput{
			{AccountUserID: "acus_o", Role: "owner"}, {AccountUserID: "acus_a", Role: "admin"},
			{AccountUserID: "acus_m", Role: "member"}, {AccountUserID: "acus_v", Role: "viewer"},
		}, owner)

		require.Nil(t, apiErr)
		assert.Equal(t, map[string]constants.ParticipantRole{
			"acus_o": constants.ParticipantRoleOwner, "acus_a": constants.ParticipantRoleAdmin,
			"acus_m": constants.ParticipantRoleMember, "acus_v": constants.ParticipantRoleViewer,
		}, roles)
	})
}

func TestApplyParticipantRole_RecordsTheParticipantsRoleChange(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	acus := "acus_admin"
	before := &domain.ConversationParticipant{ID: "cvpt_admin", ConversationID: "cv_team", AccountUserID: &acus, Role: string(constants.ParticipantRoleMember)}
	after := *before
	after.Role = string(constants.ParticipantRoleAdmin)

	partRepo := repositorymock.NewMockParticipantRepo(ctrl)
	gomock.InOrder(
		partRepo.EXPECT().SetRole(gomock.Any(), "cv_team", acus, "admin").Return(nil),
		partRepo.EXPECT().GetByID(gomock.Any(), "cvpt_admin", "cv_team").Return(&after, nil),
	)
	outbox := &fakeOutboxRepo{}
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewParticipantRepo().Return(partRepo).AnyTimes()
	factory.EXPECT().NewOutboxRepo().Return(outbox).AnyTimes()

	apiErr := applyParticipantRole(identityCtx("messaging:create"), factory, "cv_team", before, constants.ParticipantRoleAdmin)

	require.Nil(t, apiErr)
	require.Len(t, outbox.inputs, 1)
	var event struct {
		Action       string `json:"action"`
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
		Changes      []struct {
			Field    string `json:"field"`
			OldValue string `json:"old_value"`
			NewValue string `json:"new_value"`
		} `json:"changes"`
	}
	require.NoError(t, json.Unmarshal(outbox.inputs[0].Payload.Data, &event))
	assert.Equal(t, "update", event.Action)
	assert.Equal(t, string(constants.ObjectTypeConversationParticipant), event.ResourceType)
	assert.Equal(t, "cvpt_admin", event.ResourceID)
	require.Len(t, event.Changes, 1)
	assert.Equal(t, "role", event.Changes[0].Field)
	assert.Equal(t, "member", event.Changes[0].OldValue)
	assert.Equal(t, "admin", event.Changes[0].NewValue)
}

func TestCreateConversation_DirectMessageTakesNoRoles(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	notifRepo.EXPECT().ResolveAccountUserID(gomock.Any(), testUserID, testAccountID).Return("acus_caller", nil)
	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewNotificationRepo().Return(notifRepo).AnyTimes()
	svc := NewConversationSvc(factory, nil, nil, "", nil, nil, "")

	_, apiErr := svc.CreateConversation(identityCtx("messaging:create"), domain.CreateConversationInput{
		Type:                      string(constants.ConversationTypeDM),
		ParticipantAccountUserIDs: []string{"acus_other"},
		Participants:              []domain.ParticipantRoleInput{{AccountUserID: "acus_other", Role: "admin"}},
	})

	require.NotNil(t, apiErr)
	assert.Equal(t, apierror.ErrorCodeParameterInvalid, apiErr.Code)
	assert.Equal(t, "participants", apiErr.Param)
}
