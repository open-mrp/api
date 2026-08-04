package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleAttributeID = "at_rf1w295jt5ia"
const SampleAttributeValue = "Premium"

// A selectable value within a property, such as `Red` for a `Color` property.
//
// Attributes are assigned to items to classify them.
type Attribute struct {
	// Attribute ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=attribute"`
	// The selectable value this attribute represents, such as `Red` for a `Color` property or `Large` for a `Size` property.
	Value string `json:"value" validate:"required"`
	// Swatch color used to display this attribute in the UI.
	//
	// The named colors are arbitrary display choices; `default` is a neutral fallback used when no specific swatch applies.
	ColorCode constants.Color `json:"color" validate:"required"`
	// Position of this attribute relative to its siblings within the property, starting at `1`.
	//
	// Positions are kept contiguous: creating, reordering, or deleting an attribute automatically shifts its siblings.
	SortOrder int32 `json:"sort_order"`
	// The property this attribute belongs to.
	//
	// Populated only when the attribute is returned under an item's `attributes` list.
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
	Property: &Property{
		ID:        SamplePropertyID,
		Object:    constants.ObjectTypeProperty,
		Name:      SamplePropertyName,
		CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
		UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
	},
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Attribute) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleAttribute)
}
