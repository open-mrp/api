package manufacturingmethodep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
	"github.com/open-mrp/apikit/object"
)

// Request to read an item's production flow.
type GetProductionFlowRequest struct {
	object.ListRequest
	// Item ID.
	ItemID string `path:"id" validate:"required"`
}

// Returns every manufacturing method involved in making an item, from raw materials to the item itself.
//
// Read-only, built from the active methods: the item's own method first, then the methods of the materials it makes (`supply: make`), level by level. Materials that are stocked or bought end the flow. Empty when the item has no active method.
type GetProductionFlowEndpoint struct{}

func (e *GetProductionFlowEndpoint) Materialize() *apiendpoint.APIEndpoint[*GetProductionFlowRequest, *apiresource.List[apiresource.ManufacturingMethod]] {
	return &apiendpoint.APIEndpoint[*GetProductionFlowRequest, *apiresource.List[apiresource.ManufacturingMethod]]{
		Title:             "Get Production Flow",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/catalog/items/{id}/production-flow",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeManufacturingMethod,
		ServiceHandler: func(svc any) func(ctx context.Context, req *GetProductionFlowRequest) (*apiresource.List[apiresource.ManufacturingMethod], *apierror.APIError) {
			return svc.(ManufacturingMethodSvc).GetItemProductionFlow
		},
		IncludeConfig: methodIncludes(),
	}
}
