package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

// Product lines accessible to a customer.
type CustomerProductLineAccess struct {
	// The customer whose product line access this record describes.
	Customer *Customer `json:"customer" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=customer_product_line_access"`
	// Product lines accessible to this customer.
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
