package resourceregistry

import (
	"context"

	"github.com/augno/api/services/api-gateway/internal/resourceloaders"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/api-gateway/pkg/resourcekit"
	"github.com/augno/api/shared/constants"
)

func init() {
	resourcekit.Register(&resourcekit.Definition{
		ObjectType: constants.ObjectTypeConversationLink,
		Load:       resourceloaders.LoadConversationLinks,
		Subs: []resourcekit.SubField{
			// conversation is fetched by id via the chat batch loader; its own expandable sub-objects (assignee, participants, last_message, …) recurse from the conversation registration.
			{
				Key:         "conversation",
				Target:      constants.ObjectTypeConversation,
				Cardinality: resourcekit.CardinalityOnePtr,
				ExtractIDs:  extractConversationIDFromLink,
				Populate:    populateConversationOnLink,
			},
		},
	})
}

func extractConversationIDFromLink(ctx context.Context, parent any) []string {
	l := parent.(*apiresource.ConversationLink)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeConversationLink, l.ID, "conversation_id")
	if id == "" {
		return nil
	}
	return []string{id}
}

func populateConversationOnLink(ctx context.Context, parent any, loaded map[string]any) {
	l := parent.(*apiresource.ConversationLink)
	id, _ := resourcekit.GetLoadMeta(ctx).GetString(constants.ObjectTypeConversationLink, l.ID, "conversation_id")
	if id == "" {
		return
	}
	if v, ok := loaded[id]; ok {
		l.Conversation = v.(*apiresource.Conversation)
	}
}
