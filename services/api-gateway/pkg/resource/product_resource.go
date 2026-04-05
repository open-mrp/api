package apiresource

import (
	"time"

	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	"github.com/augno/api/shared/constants"
	"github.com/augno/api/shared/timeutil"
)

const SampleProductID = "pr_01jm4r6700f8nwq3v5hx2d9ktp"

// Product represents a product resource with expandable item, product line, and product type.
type Product struct {
	// The unique identifier for the product.
	ID string `json:"id" validate:"required"`
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=product"`
	// Whether this product is visible on the customer portal.
	IsPortalReady bool `json:"is_portal_ready"`
	// The product type.
	ProductType *ProductType `json:"product_type" expandable:"true"`
	// The product line this product belongs to.
	ProductLine *ProductLine `json:"product_line" expandable:"true"`
	// The underlying item for this product.
	Item *Item `json:"item" expandable:"true"`
	// The timestamp when the product was created.
	CreatedAt time.Time `json:"created_at" validate:"required"`
	// The timestamp when the product was last updated.
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

// ValidateProductsResponse represents the response for the validate products endpoint.
type ValidateProductsResponse struct {
	// The resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=map"`
	// The validated products keyed by the original map key.
	Products map[string]*Product `json:"products" validate:"required"`
}

var SampleValidateProductsResponse = &ValidateProductsResponse{
	Object:   constants.ObjectTypeMap,
	Products: map[string]*Product{"0": SampleProduct},
}

func (*ValidateProductsResponse) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(SampleValidateProductsResponse)
}
