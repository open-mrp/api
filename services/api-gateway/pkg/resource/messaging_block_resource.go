package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleMessagingBlockID = "mgbk_4azq38nghg78"

// A block one account user has placed on another.
//
// While the block stands, neither of the two can start a direct message with the other or post in an existing one, whichever of them created it. Group conversations and customer cases are unaffected.
type MessagingBlock struct {
	// Block ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=messaging_block"`
	// The account user who was blocked.
	BlockedUser *AccountUser `json:"blocked_user" expandable:"true"`
	// When the block was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
}

var SampleMessagingBlock = &MessagingBlock{
	ID:          SampleMessagingBlockID,
	Object:      constants.ObjectTypeMessagingBlock,
	BlockedUser: SampleAccountUser,
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
}

func (*MessagingBlock) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMessagingBlock)
}
