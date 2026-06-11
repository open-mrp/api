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
	// The selectable value this attribute represents, such as `Red` for a `Color` property or `Large` for a `Size` property.
	Value string `json:"value" validate:"required"`
	// Swatch color used to display this attribute in the UI.
	//
	// One of `blue`, `brown`, `gray`, `green`, `orange`, `pink`, `purple`, `red`, `yellow`, or `default` (a neutral fallback color).
	ColorCode constants.Color `json:"color" validate:"required"`
	// Position of this attribute relative to its siblings within the property, ascending.
	//
	// Lower values sort first.
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
