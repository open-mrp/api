package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleMaterialID = "ml_01jm4r6700f8nwq3v5hx2d9ktp"

// Material represents a material entity, extending an item with order point and lead time.
type Material struct {
	// The unique identifier for the material.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=material"`
	// The item this material extends.
	Item *Item `json:"item" expandable:"true"`
	// The order point quantity for this material.
	OrderPoint *QuantityInfo `json:"order_point"`
	// The lead time quantity for this material.
	LeadTime *QuantityInfo `json:"lead_time"`
	// The timestamp when the material was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the material was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleMaterial = &Material{
	ID:         SampleMaterialID,
	Object:     constants.ObjectTypeMaterial,
	Item:       SampleItem,
	OrderPoint: SampleQuantityInfo,
	LeadTime:   SampleQuantityInfo,
	CreatedAt:  timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:  timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Material) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleMaterial)
}
