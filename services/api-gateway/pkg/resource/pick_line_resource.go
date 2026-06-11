package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SamplePickLineID = "pkln_0170b1525f1c9843b22d914426"

// PickLine is a pick line resource.
type PickLine struct {
	// Pick line ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=pick_line"`
	// Quantity actually picked for this line.
	//
	// May be less than `ordered_quantity` when stock is short or the line is only partially fulfilled.
	Quantity *Quantity `json:"quantity" validate:"required"`
	// Quantity requested by the originating sales order line for this pick line.
	OrderedQuantity *Quantity `json:"ordered_quantity" validate:"required"`
	// Associated sales order line.
	//
	// Expandable via include[]=lines.sales_order_line.
	SalesOrderLine *SalesOrderLine `json:"sales_order_line" expandable:"true"`
	// Timestamp when the line was packed.
	//
	// `null` until the line has been packed.
	PackedAt *time.Time `json:"packed_at"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SamplePickLine = &PickLine{
	ID:              SamplePickLineID,
	Object:          constants.ObjectTypePickLine,
	Quantity:        SampleQuantity,
	OrderedQuantity: SampleQuantity,
	SalesOrderLine:  SampleSalesOrderLine,
	CreatedAt:       timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:       timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*PickLine) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SamplePickLine)
}
