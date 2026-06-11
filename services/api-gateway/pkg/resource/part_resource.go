package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePartID = "pt_018d7bab53e864351f4c693a21"
const SamplePartSKU = "BRG-6204-2RS"

// A part in the account's catalog: a component used in production.
//
// Part-level data such as the SKU, description, category, pricing, and attributes lives on the underlying `item`.
type Part struct {
	// Part ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=part"`
	// The underlying inventory item this part record represents.
	Item *Item `json:"item" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SamplePart = &Part{
	ID:        SamplePartID,
	Object:    constants.ObjectTypePart,
	Item:      SampleItem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Part) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePart)
}
