package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// The product lines directly accessible to a customer.
//
// Determines which product lines (and their products) the customer can browse and order. Direct access granted here combines with any access the customer inherits through its type group or pricing groups.
type CustomerProductLineAccess struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_product_line_access"`
	// The customer whose product line access this record describes.
	//
	// There is at most one access record per customer, so this also identifies the record.
	Customer *Customer `json:"customer" validate:"required"`
	// Product lines this customer has been granted direct access to.
	//
	// Only product lines your account owns can be granted; the shared system product lines never appear here.
	ProductLines *List[ProductLine] `json:"product_lines" validate:"required"`
	// When the relationship with this customer was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When the relationship with this customer was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleCustomerProductLineAccess = &CustomerProductLineAccess{
	Object:       constants.ObjectTypeCustomerProductLineAccess,
	Customer:     SampleCustomer,
	ProductLines: NewList([]ProductLine{*SampleProductLine}, PageInfo{}),
	CreatedAt:    timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:    timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*CustomerProductLineAccess) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCustomerProductLineAccess)
}
