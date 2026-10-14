package participantep

import (
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// conversationIncludeConfig exposes the conversation ?include= keys on the participant-action
// endpoints (add participant, set role), which return the updated Conversation.
func conversationIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeConversation,
		Fields: []string{
			"participants",
			"related.record",
			"related.continues",
			"last_message",
			"last_message.sender",
			"last_message.author",
			"last_message.related.record",
			"last_message.attachments",
		},
	})
}
