package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePartID = "it_02kn5s7811g9qwce7cizr4e0mq"
const SamplePartSKU = "BRG-6204-2RS"

// Part resource.
type Part struct {
	// Part ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=part"`
	// SKU.
	SKU string `json:"sku" validate:"required"`
	// Description.
	Description *string `json:"description"`
	// Notes.
	Notes *string `json:"notes"`
	// Item category.
	Category *ItemCategory `json:"category" expandable:"true"`
	// Unit value rate.
	UnitValue *Rate `json:"unit_value" expandable:"true"`
	// Unit cost rate.
	UnitCost *Rate `json:"unit_cost" expandable:"true"`
	// Burn rate.
	BurnRate *Rate `json:"burn_rate" expandable:"true"`
	// Attributes.
	Attributes *List[Attribute] `json:"attributes"`
	// Whether the part has unsaved changes.
	IsDirty bool `json:"is_dirty"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var samplePartDescription = "Sealed ball bearing, 20mm bore, 47mm OD"

var SamplePart = &Part{
	ID:          SamplePartID,
	Object:      constants.ObjectTypePart,
	SKU:         SamplePartSKU,
	Description: &samplePartDescription,
	Notes:       nil,
	Category:    SampleItemCategory,
	UnitValue:   SampleRate,
	UnitCost:    SampleRate,
	BurnRate:    SampleRate,
	Attributes:  NewList([]Attribute{*SampleAttribute}, PageInfo{}),
	IsDirty:     false,
	CreatedAt:   timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:   timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Part) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePart)
}
