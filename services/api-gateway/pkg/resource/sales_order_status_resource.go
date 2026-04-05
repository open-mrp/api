package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSalesOrderStatusID = "orss_01jm4r6700f8nwq3v5hx2d9ktp"

var SampleSalesOrderStatusCode = constants.SalesOrderStatusCodeEstimate

const SampleSalesOrderStatusName = "Estimate"

// SalesOrderStatus represents a sales order status lookup value.
type SalesOrderStatus struct {
	// The unique identifier for the sales order status.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_status"`
	// The machine-readable code for this status.
	Code constants.SalesOrderStatusCode `json:"code" validate:"required"`
	// The display name of the sales order status.
	Name string `json:"name" validate:"required"`
	// The owner of this resource.
	Owner *Owner `json:"owner" expandable:"true"`
	// When this sales order status was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this sales order status was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleSalesOrderStatus = &SalesOrderStatus{
	ID:        SampleSalesOrderStatusID,
	Object:    constants.ObjectTypeSalesOrderStatus,
	Code:      SampleSalesOrderStatusCode,
	Name:      SampleSalesOrderStatusName,
	Owner:     SampleOwnerSystem,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*SalesOrderStatus) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleSalesOrderStatus)
}
