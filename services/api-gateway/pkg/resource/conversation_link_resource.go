package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleConversationLinkID = "cvlk_cjaz69kz9dvn"

// A record a conversation concerns, such as an order, invoice, shipment or customer.
//
// Links sit alongside the record the conversation is about (its `related.record`), so one conversation can reference several records. Listing conversations by record matches both, which is what surfaces a conversation on the record's own page.
type ConversationLink struct {
	// Conversation link ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=conversation_link"`
	// The conversation this link belongs to.
	Conversation *Conversation `json:"conversation" expandable:"true"`
	// The linked record.
	Record *Record `json:"record" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

var SampleConversationLink = &ConversationLink{
	ID:           SampleConversationLinkID,
	Object:       constants.ObjectTypeConversationLink,
	Conversation: SampleConversation,
	Record:       &Record{ID: SampleSalesOrderID, Object: constants.ObjectTypeRecord, Type: constants.RecordTypeSalesOrder, Number: new("SO-1042"), Status: new("open"), Metadata: map[string]string{}},
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*ConversationLink) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleConversationLink)
}
