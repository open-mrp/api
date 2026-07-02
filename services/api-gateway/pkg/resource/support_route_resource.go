package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSupportRouteID = "spru_018e88072d1320808dc9ccc03"

// A support route designates the group conversation that handles a relationship's inbound support.
//
// Its group conversation's participants become the deterministic recipients seated on a customer's support thread. The scope is `relation_account_id`: null is the account-level default for any customer; a concrete account id is a per-relation override that wins over the default.
type SupportRoute struct {
	// Support route ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=support_route"`
	// The customer account this route overrides for.
	//
	// When null, this route is the account-level default applied to any customer without a per-relation override.
	RelationAccount *Entity `json:"relation_account"`
	// The group conversation whose participants receive this relationship's support.
	GroupConversation *Entity `json:"group_conversation" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleSupportRoute = &SupportRoute{
	ID:                SampleSupportRouteID,
	Object:            constants.ObjectTypeSupportRoute,
	GroupConversation: NewEntity(SampleConversationID, constants.ObjectTypeConversation, nil, nil),
	CreatedAt:         timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:         timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SupportRoute) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSupportRoute)
}
