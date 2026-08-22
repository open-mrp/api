package productlineep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to create a product line.
type CreateProductLineRequest struct {
	// Display name of the product line.
	//
	// Must be unique among the product lines visible to your account, including the shared system lines; a duplicate name returns a conflict error.
	Name string `json:"name" validate:"required,max=255"`
	// ID of the unit group to associate with this product line.
	//
	// The unit group determines the set of units available to products in this product line. It must be a unit group your account owns or one of the shared system unit groups.
	UnitGroupID string `json:"unit_group_id" validate:"required"`
	// Default commission policy for products in this product line.
	//
	// - `commission_exempt`: no commission applies to these products.
	// - `commission_applied`: commission applies to these products, unless overridden elsewhere.
	CommissionPolicy constants.CommissionPolicy `json:"commission_policy" validate:"required"`
	// The lot products in this line are made in — a doff, a pallet.
	//
	// Sizes the campaigns a production schedule plans, and defaults the quantity when a batch is added to a production run. The unit is part of the value, since 60 pairs and 60 eaches are different lots, and it must belong to the unit group given by `unit_group_id`. The value must be greater than zero. Leave this out for a line with no lot convention, and planning falls back to the lot of the line an item feeds into, and then to the account-wide default lot size.
	DefaultLot field.Optional[apirequest.QuantityInput] `json:"default_lot,omitzero"`
	// How products in this line are produced when they do not say for themselves.
	//
	// - `make_to_stock`: built to the forecast, holding a safety stock against its variability.
	// - `make_to_order`: built only against orders already on the book, holding no buffer.
	FulfillmentPolicy field.Optional[constants.FulfillmentPolicy] `json:"fulfillment_policy,omitzero"`
	// Default freight policy for products in this product line.
	//
	// - `free_freight`: these products do not incur a freight charge.
	// - `billed_freight`: freight is billed for these products, unless overridden elsewhere.
	FreightPolicy constants.FreightPolicy `json:"freight_policy" validate:"required"`
}

var sampleCreateProductLineRequest = &CreateProductLineRequest{
	Name:             apiresource.SampleProductLineName,
	UnitGroupID:      apiresource.SampleUnitGroupID,
	CommissionPolicy: constants.CommissionPolicyExempt,
	FreightPolicy:    constants.FreightPolicyBilled,
}

func (*CreateProductLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateProductLineRequest)
}

// Creates a product line owned by your account.
//
// The new line starts with no products; assign products to it by setting their product line. Customers and account groups can only be granted access to lines your account owns, so this is the starting point for scoping a customer's catalog.
type CreateProductLineEndpoint struct{}

func (e *CreateProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductLineRequest, *apiresource.ProductLine] {
	return (&apiendpoint.APIEndpoint[*CreateProductLineRequest, *apiresource.ProductLine]{
		Title:               "Create Product Line",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/catalog/product-lines",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductLines, Action: types.ActionCreate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeProductLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
			return svc.(ProductLineSvc).CreateProductLine
		},
		LocationFunc: func(resp *apiresource.ProductLine) string {
			return "/v1/catalog/product-lines/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductLine,
			Fields:     []string{"owner", "owner.account", "unit_group", "default_lot", "default_lot.unit"},
		}),
	})
}
