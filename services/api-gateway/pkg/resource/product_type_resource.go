package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleProductTypeID = "prty_01ddca85eedfb6b101a3c2f379"
const SampleProductTypeName = "Sale"
const SampleProductTypeCode = "sale"

// ProductType classifies how a product behaves on orders and invoices.
type ProductType struct {
	// Product type ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=product_type"`
	// Human-readable name of the product type, unique across product types.
	Name string `json:"name" validate:"required"`
	// Stable machine-readable code identifying the kind of product type.
	//
	// - `sale`: a standard sellable product.
	// - `service`: a non-physical service line, such as labor or installation.
	// - `shipping`: a shipping charge applied to an order.
	// - `credit`: a credit applied against an order or invoice.
	// - `return`: a returned product (RMA).
	// - `tax`: a tax line.
	//
	// Products reference their product type by this code, and the code can be used in place of the ID when retrieving a product type.
	Code constants.ProductTypeCode `json:"code" validate:"required"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProductType = &ProductType{
	ID:        SampleProductTypeID,
	Object:    constants.ObjectTypeProductType,
	Name:      SampleProductTypeName,
	Code:      SampleProductTypeCode,
	CreatedAt: timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt: timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*ProductType) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProductType)
}
