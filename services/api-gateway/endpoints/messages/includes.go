package messageep

import (
	"github.com/open-mrp/api/shared/constants"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// messageIncludeFields is the whitelist of ?include= keys exposed by the message endpoints. sender,
// author, resource and attachments are stashed inline by the service; conversation and reply_to are
// fetched by id via the chat batch loaders.
var messageIncludeFields = []string{
	"sender",
	"author",
	"resource",
	"attachments",
	"attachments.resource",
	"conversation",
	"conversation.participants",
	"conversation.last_message",
	"reply_to",
	"reply_to.sender",
	"reply_to.author",
	"reply_to.attachments",
	"agent_run",
}

func messageIncludeConfig() *apiendpoint.IncludeConfig {
	return apiendpoint.IncludesFor(apiendpoint.IncludesParams{
		ObjectType: constants.ObjectTypeChatMessage,
		Fields:     messageIncludeFields,
	})
}
