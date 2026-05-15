package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleMaterialID = "ml_01jm4r6700f8nwq3v5hx2d9ktp"

// Material with order point and lead time.
type Material struct {
	// Material ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=material"`
	// Item this material extends.
	Item *Item `json:"item" expandable:"true"`
	// Order point quantity.
	OrderPoint *Quantity `json:"order_point"`
	// Lead time quantity.
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
