package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleConversationLinkID = "cvlk_cjaz69kz9dvn"

// A reference from a conversation to a business record it concerns, such as an order, invoice, shipment, or customer.
//
// Links sit alongside the conversation's primary `topic` anchor, so one thread can reference several records. Listing conversations by business record matches the topic anchor and these links alike, which is what surfaces a conversation on the record's own page.
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
