package apiresource

import (
	"time"

	"github.com/open-mrp/api/shared/constants"
	apiexample "github.com/open-mrp/apikit/example"
	"github.com/open-mrp/apikit/timeutil"
)

const SamplePropertyID = "pp_fhnnvtt3q3ov"
const SamplePropertyName = "Color"

// A way items vary, such as `Size`, with the attributes it can take.
type Property struct {
	// Property ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=property"`
	// Display name, such as `Size`.
	//
	// Unique (case-insensitive) among your account's properties.
	Name string `json:"name" validate:"required"`
	// The property's attributes, in `sort_order`.
	//
	// Returns the first 10. When there are more, `page_info.next_page_url` fetches the rest.
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
