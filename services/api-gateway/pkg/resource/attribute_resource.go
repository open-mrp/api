package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAttributeID = "at_01jm4r6700f8nwq3v5hx2d9ktp"
const SampleAttributeValue = "Premium"

// Attribute represents a value option within a property.
type Attribute struct {
	// The unique identifier for the attribute.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=attribute"`
	// The value of the attribute.
	Value string `json:"value" validate:"required"`
	// The color code of the attribute.
	ColorCode constants.Color `json:"color" validate:"required"`
	// The display order of the attribute.
	SortOrder int32 `json:"sort_order"`
	// The timestamp when the attribute was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the attribute was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleAttribute = &Attribute{
	ID:        SampleAttributeID,
	Object:    constants.ObjectTypeAttribute,
	Value:     SampleAttributeValue,
	ColorCode: constants.ColorRed,
	SortOrder: 1,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Attribute) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAttribute)
}
