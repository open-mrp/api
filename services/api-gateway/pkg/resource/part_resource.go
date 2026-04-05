package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePartID = "it_02kn5s7811g9qwce7cizr4e0mq"
const SamplePartSKU = "BRG-6204-2RS"

// Part represents a part (an item specialization).
type Part struct {
	// The unique identifier for the part (item ID).
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=part"`
	// The stock keeping unit code.
	SKU string `json:"sku" validate:"required"`
	// A description of the part.
	Description *string `json:"description"`
	// Additional notes about the part.
	Notes *string `json:"notes"`
	// The item category.
	Category *ItemCategory `json:"category" expandable:"true"`
	// The unit value rate for this part.
	UnitValue *Rate `json:"unit_value" expandable:"true"`
	// The unit cost rate for this part.
	UnitCost *Rate `json:"unit_cost" expandable:"true"`
	// The burn rate for this part.
	BurnRate *Rate `json:"burn_rate" expandable:"true"`
	// The attributes assigned to this part.
	Attributes *List[Attribute] `json:"attributes"`
	// Whether the part has unsaved changes.
	IsDirty bool `json:"is_dirty"`
	// The timestamp when the part was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the part was last updated.
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
