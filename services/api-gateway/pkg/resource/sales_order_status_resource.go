package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleSalesOrderStatusID = "orss_017a18cc8a4e6dfbc61f11f5a3"

var SampleSalesOrderStatusCode = constants.SalesOrderStatusCodeEstimate

const SampleSalesOrderStatusName = "Estimate"

// Sales order status lookup value.
type SalesOrderStatus struct {
	// Sales order status ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=sales_order_status"`
	// Machine-readable status code.
	//
	// - `estimate`: a draft quote that has not yet been committed.
	// - `issued`: the order has been issued and is being fulfilled.
	// - `fulfilled`: the order has been completed and closed.
	Code constants.SalesOrderStatusCode `json:"code" validate:"required"`
	// Display name.
	Name string `json:"name" validate:"required"`
	// The owner of this status value; sales order statuses are platform-defined.
	Owner *Owner `json:"owner" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
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
