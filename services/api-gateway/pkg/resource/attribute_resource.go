package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAttributeID = "at_01c9493ec0c46bb0ed12708ae4"
const SampleAttributeValue = "Premium"

// Value option within a property.
type Attribute struct {
	// Attribute ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=attribute"`
	// Attribute value.
	Value string `json:"value" validate:"required"`
	// Color code.
	ColorCode constants.Color `json:"color" validate:"required"`
	// Display order.
	SortOrder int32 `json:"sort_order"`
	// Property this attribute belongs to (set when the attribute is returned under item.attributes).
	Property *Property `json:"property"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
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
