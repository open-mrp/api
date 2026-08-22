package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestResolveSenders_AnonymizationByViewerRelation pins the top correctness risk: on an external case,
// a customer-relation viewer must see every vendor-side (staff/agent) author collapsed to the single
// branded "Customer Service" alias with the real author stripped, while internal viewers always retain
// the real author. A regression here silently leaks the operator behind "Customer Service" to the customer.
func TestResolveSenders_AnonymizationByViewerRelation(t *testing.T) {
	const (
		conversationID = "cv_anon1234"
		accountID      = "ac_anon1234"
		participantID  = "cvpt_author1234"
		authorAcus     = "acus_author1234"
	)

	// The author is a real internal staff member.
	author := &domain.ConversationParticipant{
		ID:              participantID,
		ConversationID:  conversationID,
		AccountID:       accountID,
		ParticipantType: string(constants.ParticipantTypeUser),
		AccountUserID:   strPtr(authorAcus),
	}

	newMessage := func() *domain.Message {
		return &domain.Message{
			ID:                  "mg_anon1234",
			ConversationID:      conversationID,
			AccountID:           accountID,
			SenderParticipantID: strPtr(participantID),
		}
	}

	newSvc := func(t *testing.T) *conversationSvcImpl {
		t.Helper()
		ctrl := gomock.NewController(t)

		participantRepo := repositorymock.NewMockParticipantRepo(ctrl)
		participantRepo.EXPECT().
			ListAll(gomock.Any(), conversationID).
			Return([]*domain.ConversationParticipant{author}, nil).
			AnyTimes()

		factory := factorymock.NewMockRepoFactory(ctrl)
		factory.EXPECT().NewParticipantRepo().Return(participantRepo).AnyTimes()

		return &conversationSvcImpl{repoFactory: factory}
	}

	t.Run("customer viewer sees the branded alias, never the real author", func(t *testing.T) {
		svc := newSvc(t)
		msg := newMessage()

		svc.resolveSenders(context.Background(), conversationID, accountID, []*domain.Message{msg}, true /* viewerIsRelation */)

		assert.Nil(t, msg.SenderAccountUserID, "the real staff author must be stripped for a customer-relation viewer")
		require.NotNil(t, msg.SenderAlias)
		assert.Equal(t, "Customer Service", *msg.SenderAlias, "the customer sees the branded party")
	})

	t.Run("internal viewer always sees the real author", func(t *testing.T) {
		svc := newSvc(t)
		msg := newMessage()

		svc.resolveSenders(context.Background(), conversationID, accountID, []*domain.Message{msg}, false /* viewerIsRelation */)

		require.NotNil(t, msg.SenderAccountUserID, "internal viewers must retain the real author")
		assert.Equal(t, authorAcus, *msg.SenderAccountUserID)
		assert.Nil(t, msg.SenderAlias, "internal viewers get no branded alias")
	})
}

// TestFilterCustomerVisibleParticipants pins that a customer viewing their portal support case never
// receives the vendor-side roster: staff and agent participants are stripped so neither their identities
// nor their head count (the participant array's length) leak. Only the customer's own rows remain. This is
// the participant-roster counterpart to the message-author anonymization in resolveSenders.
func TestFilterCustomerVisibleParticipants(t *testing.T) {
	const conversationID = "cv_filter1234"

	part := func(id string, pType constants.ParticipantType) *domain.ConversationParticipant {
		return &domain.ConversationParticipant{ID: id, ConversationID: conversationID, ParticipantType: string(pType)}
	}

	participants := []*domain.ConversationParticipant{
		part("cvpt_cust1", constants.ParticipantTypeCustomer),
		part("cvpt_staff1", constants.ParticipantTypeUser),
		part("cvpt_staff2", constants.ParticipantTypeUser),
		part("cvpt_agent1", constants.ParticipantTypeAgent),
	}

	got := filterCustomerVisibleParticipants(participants)

	require.Len(t, got, 1, "only the customer's own participant survives; the vendor head count must not leak")
	assert.Equal(t, "cvpt_cust1", got[0].ID)
	for _, p := range got {
		assert.Equal(t, string(constants.ParticipantTypeCustomer), p.ParticipantType, "no vendor-side participant may remain")
	}
}

func strPtr(s string) *string { return &s }
