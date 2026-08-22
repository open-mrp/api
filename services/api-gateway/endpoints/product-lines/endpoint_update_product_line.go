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

// Request to partially update a product line.
type UpdateProductLineRequest struct {
	// Product line ID.
	ProductLineID string `path:"id" validate:"required"`
	// Display name of the product line.
	//
	// Must be unique among the product lines visible to your account, including the shared system lines; a duplicate name returns a conflict error.
	Name field.Optional[string] `json:"name,omitzero" validate:"omitempty,max=255"`
	// Default commission policy for products in this product line.
	//
	// - `commission_exempt`: no commission applies to these products.
	// - `commission_applied`: commission applies to these products, unless overridden elsewhere.
	CommissionPolicy field.Optional[constants.CommissionPolicy] `json:"commission_policy,omitzero"`
	// Default freight policy for products in this product line.
	//
	// - `free_freight`: these products do not incur a freight charge.
	// - `billed_freight`: freight is billed for these products, unless overridden elsewhere.
	FreightPolicy field.Optional[constants.FreightPolicy] `json:"freight_policy,omitzero"`
	// The lot products in this line are made in — a doff, a pallet.
	//
	// Sizes the campaigns a production schedule plans, and defaults the quantity when a batch is added to a production run. The unit is part of the value, since 60 pairs and 60 eaches are different lots, and it must belong to the line's unit group — the new one when `unit_group_id` changes in the same request. Send `null` to remove the line's lot convention, after which planning falls back to the lot of the line an item feeds into, and then to the account-wide default lot size.
	DefaultLot field.Clearable[apirequest.QuantityInput] `json:"default_lot,omitzero"`
	// How products in this line are produced when they do not say for themselves.
	//
	// - `make_to_stock`: built to the forecast, holding a safety stock against its variability.
	// - `make_to_order`: built only against orders already on the book, holding no buffer.
	//
	// Clearing it returns the line's products to the account default.
	FulfillmentPolicy field.Clearable[constants.FulfillmentPolicy] `json:"fulfillment_policy,omitzero"`
	// ID of the unit group to associate with this product line.
	//
	// The unit group determines the set of units available to products in this product line. It must be a unit group your account owns or one of the shared system unit groups. A lot already stored on the line is not rechecked when the group changes, so send `default_lot` alongside to keep the two consistent.
	UnitGroupID field.Optional[string] `json:"unit_group_id,omitzero" validate:"omitempty"`
}

var sampleUpdateProductLineName = "Updated Product Line"

var sampleUpdateProductLineRequest = &UpdateProductLineRequest{
	Name:             field.Some(sampleUpdateProductLineName),
	CommissionPolicy: field.Some(constants.CommissionPolicyApplied),
	FreightPolicy:    field.Some(constants.FreightPolicyBilled),
	UnitGroupID:      field.Some(apiresource.SampleUnitGroupID),
}

func (*UpdateProductLineRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateProductLineRequest)
}

// Partially updates a product line your account owns.
//
// Only the provided fields are changed. The reserved `shipping`, `service`, `credit`, and `tax` lines cannot be updated, and neither can the shared system lines, which belong to no single account.
type UpdateProductLineEndpoint struct{}

func (e *UpdateProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateProductLineRequest, *apiresource.ProductLine] {
	return (&apiendpoint.APIEndpoint[*UpdateProductLineRequest, *apiresource.ProductLine]{
		Title:               "Update Product Line",
		Method:              http.MethodPatch,
		Route:               "/v1/catalog/product-lines/{id}",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductLines, Action: types.ActionUpdate}},
		Preview:             true,
		ObjectType:          constants.ObjectTypeProductLine,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateProductLineRequest) (*apiresource.ProductLine, *apierror.APIError) {
			return svc.(ProductLineSvc).UpdateProductLine
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductLine,
			Fields:     []string{"owner", "owner.account", "unit_group", "default_lot", "default_lot.unit"},
		}),
	})
}
