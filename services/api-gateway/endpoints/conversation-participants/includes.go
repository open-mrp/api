package participantep

import (
	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	"github.com/augno/api/shared/constants"
)

// conversationIncludeConfig exposes the conversation ?include= keys on the participant-action
// endpoints (add participant, set role), which return the updated Conversation.
func conversationIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeConversation,
		Fields: []string{
			"participants",
			"topic",
			"last_message",
			"last_message.sender",
			"last_message.author",
			"last_message.resource",
			"last_message.attachments",
		},
	})
}
