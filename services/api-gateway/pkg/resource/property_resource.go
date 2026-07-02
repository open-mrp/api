package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePropertyID = "pp_01e21344878064372f69e67093"
const SamplePropertyName = "Color"

// A named characteristic used to classify items, such as `Color` or `Size`.
//
// Each property defines a set of attributes — the selectable values (e.g. `Red`, `Blue`) that can be assigned to items.
type Property struct {
	// Property ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=property"`
	// Display name of the property, such as `Color` or `Size`.
	Name string `json:"name" validate:"required"`
	// Attributes belonging to this property.
	Attributes *List[Attribute] `json:"attributes" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last update timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProperty = &Property{
	ID:         SamplePropertyID,
	Object:     constants.ObjectTypeProperty,
	Name:       SamplePropertyName,
	Attributes: NewList([]Attribute{*SampleAttribute}, PageInfo{}),
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Property) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProperty)
}
