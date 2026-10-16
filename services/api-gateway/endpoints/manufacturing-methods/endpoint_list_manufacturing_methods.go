package manufacturingmethodep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apirequest "github.com/open-mrp/api/services/api-gateway/pkg/request"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list manufacturing methods.
type ListManufacturingMethodsRequest struct {
	object.ListRequest
	apirequest.CreatedRange
	// Only return methods that make one of these items.
	ItemIDs []string `query:"item_ids" validate:"omitempty,max=100"`
	// Only return methods with one of these statuses: `draft`, `active` or `archived`. Omit to return all statuses.
	Statuses []constants.ManufacturingMethodStatus `query:"statuses"`
	// Only return methods with an operation at one of these work centers.
	WorkCenterIDs []string `query:"work_center_ids" validate:"omitempty,max=100"`
}

// Returns a list of manufacturing methods.
//
// Every version of every item's method is listed, newest first; filter by `item_ids[]` for one item's versions, or by `statuses[]` for the active ones.
type ListManufacturingMethodsEndpoint struct{}

func (e *ListManufacturingMethodsEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListManufacturingMethodsRequest, *apiresource.List[apiresource.ManufacturingMethod]] {
	return &apiendpoint.APIEndpoint[*ListManufacturingMethodsRequest, *apiresource.List[apiresource.ManufacturingMethod]]{
		Title:             "List Manufacturing Methods",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             methodsRoute,
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		AgentTool:         true,
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainManufacturingMethods, Action: types.ActionRead}}},
		ObjectType:        constants.ObjectTypeManufacturingMethod,
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListManufacturingMethodsRequest) (*apiresource.List[apiresource.ManufacturingMethod], *apierror.APIError) {
			return svc.(ManufacturingMethodSvc).ListManufacturingMethods
		},
		IncludeConfig: methodIncludes(),
	}
}
