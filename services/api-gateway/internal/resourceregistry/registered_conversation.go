package resourceregistry

import (
	"context"

	"github.com/open-mrp/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/api-gateway/pkg/resourcekit"
	"github.com/open-mrp/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeConversation,
		Load:       resourceloaders.LoadConversations,
		Subs: []resourcekit.SubField{
			// assignee, group, participants, topic and last_message all arrive inline on the conversation payload and are stashed by chatmap.StashConversationMeta — no extra fetch, just gated visibility.
			{Key: "assignee", Cardinality: resourcekit.CardinalityOnePtr, Populate: populateAssigneeOnConversation},
			{Key: "group", Cardinality: resourcekit.CardinalityOnePtr, Populate: populateGroupOnConversation},
			{Key: "participants", Cardinality: resourcekit.CardinalityList, Populate: populateParticipantsOnConversation},
			{Key: "topic", Cardinality: resourcekit.CardinalityOnePtr, Populate: populateTopicOnConversation},
			{
				Key:         "last_message",
				Target:      constants.ObjectTypeChatMessage,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractRefs: extractLastMessageRefs,
				Populate:    populateLastMessageOnConversation,
			},
		},
	})
}

func populateAssigneeOnConversation(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Conversation)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeConversation, c.ID, "assignee")
	if !ok {
		return
	}
	c.Assignee = v.(*apiresource.Actor)
}

func populateGroupOnConversation(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Conversation)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeConversation, c.ID, "group")
	if !ok {
		return
	}
	c.Group = v.(*apiresource.MessagingGroup)
}

func populateParticipantsOnConversation(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Conversation)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeConversation, c.ID, "participants")
	if !ok {
		return
	}
	c.Participants = v.(*apiresource.List[apiresource.ConversationParticipant])
}

func populateTopicOnConversation(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Conversation)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeConversation, c.ID, "topic")
	if !ok {
		return
	}
	c.Topic = v.(*apiresource.Entity)
}

func populateLastMessageOnConversation(ctx context.Context, parent any, _ map[string]any) {
	c := parent.(*apiresource.Conversation)
	v, ok := resourcekit.GetLoadMeta(ctx).Get(constants.ObjectTypeConversation, c.ID, "last_message")
	if !ok {
		return
	}
	c.LastMessage = v.(*apiresource.Message)
}

func extractLastMessageRefs(ctx context.Context, parent any) []any {
	c := parent.(*apiresource.Conversation)
	if c.LastMessage == nil {
		return nil
	}
	return []any{c.LastMessage}
}
