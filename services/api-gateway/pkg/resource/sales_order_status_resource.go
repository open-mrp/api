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

// A lookup value describing where a sales order is in its lifecycle, from estimate through fulfillment.
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
	// Human-readable name of the status.
	Name string `json:"name" validate:"required"`
	// Owner of this status value.
	//
	// Sales order statuses are platform-provided and shared across all accounts, so the owner is always the Augno system owner.
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
