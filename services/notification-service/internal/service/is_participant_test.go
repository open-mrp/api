package service

import (
	"context"
	"testing"

	"github.com/augno/api/services/notification-service/internal/domain"
	factorymock "github.com/augno/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/augno/api/services/notification-service/internal/domain/mock/repository"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestIsParticipant_InternalAccountUser(t *testing.T) {
	const (
		conversationID = "cv_part1234"
		vendorAccount  = "ac_vendor1234"
		userID         = "us_staff1234"
		staffAcus      = "acus_staff1234"
	)

	ctrl := gomock.NewController(t)

	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	notifRepo.EXPECT().
		ResolveAccountUserID(gomock.Any(), userID, vendorAccount).
		Return(staffAcus, nil)

	participantRepo := repositorymock.NewMockParticipantRepo(ctrl)
	participantRepo.EXPECT().
		Get(gomock.Any(), conversationID, staffAcus).
		Return(&domain.ConversationParticipant{
			Membership: string(constants.ParticipantMembershipActive),
		}, nil)

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewNotificationRepo().Return(notifRepo)
	factory.EXPECT().NewParticipantRepo().Return(participantRepo)

	svc := &conversationSvcImpl{repoFactory: factory}
	ok, apiErr := svc.IsParticipant(context.Background(), conversationID, userID, vendorAccount)
	require.Nil(t, apiErr)
	assert.True(t, ok)
}

func TestIsParticipant_RelationActorViaCustomerParticipant(t *testing.T) {
	const (
		conversationID  = "cv_part1234"
		vendorAccount   = "ac_vendor1234"
		customerAccount = "ac_customer1234"
		userID          = "us_customer1234"
	)

	ctrl := gomock.NewController(t)

	notifRepo := repositorymock.NewMockNotificationRepo(ctrl)
	notifRepo.EXPECT().
		ResolveAccountUserID(gomock.Any(), userID, vendorAccount).
		Return("", apierror.NewResourceNotFoundError("not found"))
	notifRepo.EXPECT().
		ResolveAccountUserID(gomock.Any(), userID, customerAccount).
		Return("acus_customer1234", nil)

	participantRepo := repositorymock.NewMockParticipantRepo(ctrl)
	participantRepo.EXPECT().
		List(gomock.Any(), conversationID).
		Return([]*domain.ConversationParticipant{
			{
				ParticipantType:   string(constants.ParticipantTypeCustomer),
				RelationAccountID: strPtr(customerAccount),
				Membership:        string(constants.ParticipantMembershipActive),
			},
		}, nil)

	factory := factorymock.NewMockRepoFactory(ctrl)
	factory.EXPECT().NewNotificationRepo().Return(notifRepo)
	factory.EXPECT().NewParticipantRepo().Return(participantRepo)

	svc := &conversationSvcImpl{repoFactory: factory}
	ok, apiErr := svc.IsParticipant(context.Background(), conversationID, userID, vendorAccount)
	require.Nil(t, apiErr)
	assert.True(t, ok)
}
