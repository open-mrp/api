package service

import (
	"context"
	"testing"

	factorymock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/factory"
	repomock "github.com/open-mrp/api/services/notification-service/internal/domain/mock/repository"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAgentTriggerFires_Always(t *testing.T) {
	assert.True(t, agentTriggerFires(constants.AgentTriggerPolicyAlways, nil, "anything"))
	assert.True(t, agentTriggerFires(constants.AgentTriggerPolicyAlways, nil, ""))
}

func TestAgentTriggerFires_Keyword(t *testing.T) {
	kws := []string{"forecast", "Order"}
	assert.True(t, agentTriggerFires(constants.AgentTriggerPolicyKeyword, kws, "what is the FORECAST today"), "case-insensitive keyword match")
	assert.True(t, agentTriggerFires(constants.AgentTriggerPolicyKeyword, kws, "place an order please"))
	assert.False(t, agentTriggerFires(constants.AgentTriggerPolicyKeyword, kws, "hello there"))
	assert.False(t, agentTriggerFires(constants.AgentTriggerPolicyKeyword, nil, "forecast"), "no keywords configured → never fires")
}

func TestAgentTriggerFires_Mention(t *testing.T) {
	kws := []string{"planner"}
	assert.True(t, agentTriggerFires(constants.AgentTriggerPolicyMention, kws, "hey @planner can you help"), "mention requires the @ prefix")
	assert.False(t, agentTriggerFires(constants.AgentTriggerPolicyMention, kws, "the planner is out"), "bare keyword without @ does not mention")
}

func TestAgentTriggerFires_UnknownPolicy(t *testing.T) {
	assert.False(t, agentTriggerFires(constants.AgentTriggerPolicy("bogus"), []string{"x"}, "x"))
}

func TestChatSenderUserID(t *testing.T) {
	ctx := context.Background()

	t.Run("a member sender resolves to their user", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		f := factorymock.NewMockRepoFactory(ctrl)
		notifRepo := repomock.NewMockNotificationRepo(ctrl)
		f.EXPECT().NewNotificationRepo().Return(notifRepo)
		notifRepo.EXPECT().ResolveUserIDInAccount(gomock.Any(), "acus_sender", "ac_chat").Return("us_sender", nil)

		uid, ok, apiErr := (&conversationSvcImpl{}).chatSenderUserID(ctx, f, "ac_chat", string(constants.ParticipantTypeUser), "acus_sender")
		require.Nil(t, apiErr)
		assert.True(t, ok)
		assert.Equal(t, "us_sender", uid)
	})

	t.Run("a sender who left the account wakes no agent", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		f := factorymock.NewMockRepoFactory(ctrl)
		notifRepo := repomock.NewMockNotificationRepo(ctrl)
		f.EXPECT().NewNotificationRepo().Return(notifRepo)
		notifRepo.EXPECT().ResolveUserIDInAccount(gomock.Any(), "acus_gone", "ac_chat").Return("", apierror.NewResourceNotFoundError("not found"))

		_, ok, apiErr := (&conversationSvcImpl{}).chatSenderUserID(ctx, f, "ac_chat", string(constants.ParticipantTypeUser), "acus_gone")
		require.Nil(t, apiErr)
		assert.False(t, ok)
	})

	t.Run("inbound email and customers carry no member", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		f := factorymock.NewMockRepoFactory(ctrl)
		for _, pt := range []constants.ParticipantType{constants.ParticipantTypeSystem, constants.ParticipantTypeCustomer} {
			uid, ok, apiErr := (&conversationSvcImpl{}).chatSenderUserID(ctx, f, "ac_chat", string(pt), "")
			require.Nil(t, apiErr)
			assert.True(t, ok)
			assert.Empty(t, uid)
		}
	})
}
