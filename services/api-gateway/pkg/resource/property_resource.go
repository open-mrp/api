package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePropertyID = "pp_01jm4r6700f8nwq3v5hx2d9ktp"
const SamplePropertyName = "Color"

// Property represents a property that groups attributes.
type Property struct {
	// The unique identifier for the property.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=property"`
	// The name of the property.
	Name string `json:"name" validate:"required"`
	// The attributes belonging to this property.
	Attributes *List[Attribute] `json:"attributes" expandable:"true"`
	// The timestamp when the property was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the property was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProperty = &Property{
	ID:        SamplePropertyID,
	Object:    constants.ObjectTypeProperty,
	Name:      SamplePropertyName,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Property) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProperty)
}
