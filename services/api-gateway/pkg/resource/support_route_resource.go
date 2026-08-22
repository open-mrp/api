package apiresource

import (
	"time"

	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	"github.com/open-mrp/api/shared/constants"
	"github.com/open-mrp/api/shared/timeutil"
)

const SampleSupportRouteID = "spru_m7jti68mins7"

// A support route designates the group conversation that handles a relationship's inbound support.
//
// A route is scoped by `relation_account`: the route with no relation account is the account-level default used for any customer, and a route naming a specific customer account overrides that default for that customer.
//
// When a customer opens a support thread, the route in effect for them is resolved and the group conversation's active people are seated on the new thread as its recipients. Routes are applied at that moment only, so re-pointing or clearing a route never changes who is already seated on threads that are open.
//
// The group also serves as the account's customer-service team elsewhere: its people are the ones alerted when a customer registers for access to your portal.
type SupportRoute struct {
	// Support route ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=support_route"`
	// The customer account this route overrides for.
	//
	// When there is no relation account, this route is the account-level default applied to any customer without an override of their own.
	RelationAccount *Entity `json:"relation_account"`
	// The group conversation whose participants handle this relationship's support.
	//
	// Its active people are the ones seated on a customer's support thread when the thread is opened, so who handles support is changed by editing this group's membership or by pointing the route at a different group — either way, only for threads opened afterwards.
	GroupConversation *Entity `json:"group_conversation" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleSupportRoute = &SupportRoute{
	ID:                SampleSupportRouteID,
	Object:            constants.ObjectTypeSupportRoute,
	RelationAccount:   NewEntity(SampleAccountID, constants.ObjectTypeAccount, new(SampleAccountName), nil),
	GroupConversation: NewEntity(SampleConversationID, constants.ObjectTypeConversation, nil, nil),
	CreatedAt:         timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:         timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SupportRoute) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSupportRoute)
}
