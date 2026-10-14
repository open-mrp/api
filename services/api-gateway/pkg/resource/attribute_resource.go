package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SampleAttributeID = "at_rf1w295jt5ia"
const SampleAttributeValue = "Premium"

// One value of a property, such as `Large` for `Size`: the shared descriptive layer items are classified and priced by.
type Attribute struct {
	// Attribute ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=attribute"`
	// The value, such as `Large`.
	//
	// Unique (case-insensitive) within its property.
	Value string `json:"value" validate:"required"`
	// Swatch color shown for the attribute.
	//
	// Defaults to `default`, a neutral swatch.
	ColorCode constants.Color `json:"color" validate:"required"`
	// Position within the property, starting at `1`.
	//
	// Positions stay contiguous: adding, moving or deleting an attribute shifts the others.
	SortOrder int32 `json:"sort_order"`
	// Whether the attribute can be newly assigned.
	//
	// `active` attributes can be assigned to items. `archived` attributes stay on the items that have them but can't be newly assigned.
	Status constants.ArchiveStatus `json:"status" validate:"required"`
	// The property this attribute belongs to.
	Property *Property `json:"property" expandable:"true"`
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
	Status:    constants.ArchiveStatusActive,
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
