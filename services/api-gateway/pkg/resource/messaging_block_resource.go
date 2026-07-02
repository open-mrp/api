package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleMessagingBlockID = "mgbk_01h9z8q1w2e3r4t5y6mgbk"

// A 1:1 messaging block: the caller has blocked another account user from messaging them.
type MessagingBlock struct {
	// Block ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=messaging_block"`
	// The blocked account user.
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
