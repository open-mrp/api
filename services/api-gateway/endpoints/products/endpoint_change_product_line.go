package productep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to move a product to a different product line.
type ChangeProductProductLineRequest struct {
	// Product ID.
	ProductID string `path:"id" validate:"required"`
	// ID of the product line to assign to the product.
	ProductLineID string `path:"product_line_id" validate:"required"`
}

// Moves a product to a different product line.
//
// The target product line must be one your account owns or a shared system line; anything else fails as not found. Because customer accounts are granted access to whole product lines, moving a product changes which buyers can see and order it in the customer portal, and which default commission and freight policies apply to it.
type ChangeProductProductLineEndpoint struct{}

func (e *ChangeProductProductLineEndpoint) Materialize() *apiendpoint.APIEndpoint[*ChangeProductProductLineRequest, *apiresource.Product] {
	return (&apiendpoint.APIEndpoint[*ChangeProductProductLineRequest, *apiresource.Product]{
		Title:               "Change Product Product Line",
		Method:              http.MethodPut,
		Route:               "/v1/catalog/products/{id}/product-line/{product_line_id}",
		SDKMethodKey:        "change_product_line",
		ContentType:         "application/json",
		SuccessStatusCode:   http.StatusOK,
		Public:              true,
		AgentTool:           true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainItems, Action: types.ActionUpdate}},
		Preview:             true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ChangeProductProductLineRequest) (*apiresource.Product, *apierror.APIError) {
			return svc.(ProductSvc).ChangeProductProductLine
		},
		ObjectType: constants.ObjectTypeProduct,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProduct,
			Fields:     []string{"product_line", "product_line.unit_group", "product_line.unit_group.base_unit", "product_line.unit_group.associated_units", "product_line.unit_group.associated_units.unit", "item", "item.category", "item.category.properties", "item.category.unit_group", "item.category.unit_group.base_unit", "item.category.unit_group.associated_units", "item.category.unit_group.associated_units.unit", "item.unit_value", "item.unit_cost", "item.burn_rate", "item.attributes"},
		}),
	})
}
