package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleConversationLinkID = "cvlk_01h9z8q1w2e3r4t5y6u7cvlk"

// A business-record link on a conversation: the record the conversation is about (an order, invoice, shipment, customer, …), shown prominently and usable as agent context.
type ConversationLink struct {
	// Conversation link ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=conversation_link"`
	// The conversation this link belongs to.
	Conversation *Conversation `json:"conversation" expandable:"true"`
	// The linked business record.
	Resource *Entity `json:"resource"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

var SampleConversationLink = &ConversationLink{
	ID:           SampleConversationLinkID,
	Object:       constants.ObjectTypeConversationLink,
	Conversation: SampleConversation,
	Resource:     NewEntity(SampleSalesOrderID, constants.ObjectTypeSalesOrder, new("Order #1042"), nil),
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*ConversationLink) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleConversationLink)
}
