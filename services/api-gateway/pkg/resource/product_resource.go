package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleProductID = "pr_01jm4r6700f8nwq3v5hx2d9ktp"

// Product with expandable item, product line, and product type.
type Product struct {
	// Product ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=product"`
	// Whether visible on the customer portal.
	IsPortalReady bool `json:"is_portal_ready"`
	// Product type.
	ProductType *ProductType `json:"product_type" expandable:"true"`
	// Product line.
	ProductLine *ProductLine `json:"product_line" expandable:"true"`
	// Item.
	Item *Item `json:"item" expandable:"true"`
	// Creation timestamp.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// Last updated timestamp.
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

var SampleProduct = &Product{
	ID:            SampleProductID,
	Object:        constants.ObjectTypeProduct,
	IsPortalReady: true,
	ProductType:   SampleProductType,
	ProductLine:   SampleProductLine,
	Item:          SampleItem,
	CreatedAt:     timeutil.TimestampToTime(sampleCreatedAtTimestamp),
	UpdatedAt:     timeutil.TimestampToTime(sampleUpdatedAtTimestamp),
}

func (*Product) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleProduct)
}

// ValidateProductsResponse is the response for the validate products endpoint.
type ValidateProductsResponse struct {
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=map"`
	// Validated products keyed by original map key.
	Products map[string]*Product `json:"products" validate:"required"`
}

var SampleValidateProductsResponse = &ValidateProductsResponse{
	Object:   constants.ObjectTypeMap,
	Products: map[string]*Product{"0": SampleProduct},
}

func (*ValidateProductsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleValidateProductsResponse)
}
