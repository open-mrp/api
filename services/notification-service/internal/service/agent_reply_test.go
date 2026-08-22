package service

import (
	"context"
	"testing"

	"github.com/open-mrp/api/services/notification-service/internal/domain"
	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	repositorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// PostAgentReply must silently drop a reply (without creating a message) when the agent is no longer
// an active participant, and no-op on empty input — both guards return before the message tx, so the
// (nil) tx manager is never reached.
func TestPostAgentReply_DropsWhenAgentNotActiveParticipant(t *testing.T) {
	const (
		accountID      = "ac_reply1234"
		conversationID = "cv_reply1234"
		agentConfigID  = "agdf_reply1234"
	)
	base := domain.AgentReplyInput{
		AccountID:       accountID,
		ConversationID:  conversationID,
		AgentConfigID:   agentConfigID,
		AgentRunID:      "agrn_reply1234",
		Body:            "Here's your forecast.",
		ClientMessageID: "agentreply:agrn_reply1234",
	}

	t.Run("agent no longer a participant", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		partRepo := repositorymock.NewMockParticipantRepo(ctrl)
		partRepo.EXPECT().
			GetByAgentConfigID(gomock.Any(), conversationID, agentConfigID).
			Return(nil, apierror.NewResourceNotFoundError("not a participant"))
		factory := factorymock.NewMockRepoFactory(ctrl)
		factory.EXPECT().NewParticipantRepo().Return(partRepo).AnyTimes()

		// txManager is nil: if PostAgentReply tried to create the message it would panic.
		svc := &conversationSvcImpl{repoFactory: factory}
		assert.Nil(t, svc.PostAgentReply(context.Background(), base))
	})

	t.Run("agent participant is inactive", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		partRepo := repositorymock.NewMockParticipantRepo(ctrl)
		partRepo.EXPECT().
			GetByAgentConfigID(gomock.Any(), conversationID, agentConfigID).
			Return(&domain.ConversationParticipant{ID: "cvpt_reply1234", Membership: string(constants.ParticipantMembershipRemoved)}, nil)
		factory := factorymock.NewMockRepoFactory(ctrl)
		factory.EXPECT().NewParticipantRepo().Return(partRepo).AnyTimes()

		svc := &conversationSvcImpl{repoFactory: factory}
		assert.Nil(t, svc.PostAgentReply(context.Background(), base))
	})

	t.Run("empty body is a no-op", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		factory := factorymock.NewMockRepoFactory(ctrl) // no repo calls expected — guard returns first

		svc := &conversationSvcImpl{repoFactory: factory}
		in := base
		in.Body = ""
		assert.Nil(t, svc.PostAgentReply(context.Background(), in))
	})
}
