package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleMaterialID = "ml_014613b8f7959a091d8cc0cef4"

// Material with order point and lead time.
type Material struct {
	// Material ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=material"`
	// The underlying inventory item this material record extends with material-specific fields such as order point and lead time.
	Item *Item `json:"item" expandable:"true"`
	// Reorder threshold: when on-hand stock falls to this quantity, the material should be reordered.
	//
	// Null if no reorder point is configured.
	OrderPoint *Quantity `json:"order_point"`
	// Expected time between placing an order for this material and receiving it, expressed as a quantity in a time unit (e.g. days).
	//
	// Null if not set.
	LeadTime *Quantity `json:"lead_time"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleMaterial = &Material{
	ID:         SampleMaterialID,
	Object:     constants.ObjectTypeMaterial,
	Item:       SampleItem,
	OrderPoint: SampleQuantity,
	LeadTime:   SampleQuantity,
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Material) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMaterial)
}
