package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// CustomerProductLineAccess represents the product lines accessible to a customer.
type CustomerProductLineAccess struct {
	// The customer.
	Customer *Customer `json:"customer" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_product_line_access"`
	// The product lines accessible to this customer.
	ProductLines *List[ProductLine] `json:"product_lines" validate:"required"`
	// When this record was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// When this record was last updated.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleCustomerProductLineAccess = &CustomerProductLineAccess{
	Customer: SampleCustomer,
	Object:   constants.ObjectTypeCustomerProductLineAccess,
	ProductLines: NewList([]ProductLine{
		{
			ID:     SampleProductLineID,
			Object: constants.ObjectTypeProductLine,
			Name:   SampleProductLineName,
		},
	}, PageInfo{}),
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*CustomerProductLineAccess) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleCustomerProductLineAccess)
}
