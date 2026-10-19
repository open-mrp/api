package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleSubscriptionID = "sk_7c2mq0vty3dh"

// Whether you're notified about a record.
//
// You're subscribed automatically according to your `auto_subscribe` preference. Unsubscribing stays in effect until you subscribe again.
type Subscription struct {
	// Subscription ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=subscription"`
	// The record you're notified about: a sales order, purchase order, production run, invoice, customer or supplier. Following a customer or supplier also covers their orders and invoices.
	Record *Record `json:"record" validate:"required"`
	// Why you're subscribed: `creator`, `editor`, `assignee`, `mentioned` or `manual`.
	Reason constants.SubscriptionReason `json:"reason" validate:"required"`
	// `subscribed` (you're notified) or `unsubscribed` (you aren't, and aren't subscribed again automatically).
	Status constants.RecordSubscriptionStatus `json:"status" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleSubscription = &Subscription{
	ID:        SampleSubscriptionID,
	Object:    constants.ObjectTypeSubscription,
	Record:    &Record{ID: SampleSalesOrderID, Object: constants.ObjectTypeRecord, Type: constants.RecordTypeSalesOrder, Number: new(SampleSalesOrderNumber), Status: new("issued"), Metadata: map[string]string{}},
	Reason:    constants.SubscriptionReasonCreator,
	Status:    constants.RecordSubscriptionStatusSubscribed,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Subscription) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSubscription)
}
