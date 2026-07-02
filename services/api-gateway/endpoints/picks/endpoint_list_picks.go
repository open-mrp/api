package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// ListPicksRequest is the request to list picks.
type ListPicksRequest struct {
	apiresource.PaginationRequest
	// Filter by pick status.
	//
	// - `open`: picks still in progress (not yet finished).
	// - `closed`: picks that have been finished.
	Status *string `query:"status"`
	// Filter by customer IDs.
	CustomerIDs []string `query:"customer_ids"`
	// Filter by product line IDs.
	//
	// Matches picks that contain at least one line for a product in any of the given product lines.
	ProductLineIDs []string `query:"product_line_ids"`
	// Filter by customer group IDs.
	CustomerGroupIDs []string `query:"customer_group_ids"`
	// Filter by department IDs.
	DepartmentIDs []string `query:"department_ids"`
	// Only return picks created on or after this date (`YYYY-MM-DD`).
	StartDate *string `query:"start_date"`
	// Only return picks created before this date (`YYYY-MM-DD`).
	EndDate *string `query:"end_date"`
}

// Returns a paginated list of picks.
type ListPicksEndpoint struct{}

func (e *ListPicksEndpoint) Materialize() *apiendpoint.APIEndpoint[*ListPicksRequest, *apiresource.List[apiresource.Pick]] {
	return (&apiendpoint.APIEndpoint[*ListPicksRequest, *apiresource.List[apiresource.Pick]]{
		Title:             "List Picks",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ObjectType:        constants.ObjectTypePick,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainPicks, Action: types.ActionRead},
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
			{Domain: types.PermissionDomainSuppliers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *ListPicksRequest) (*apiresource.List[apiresource.Pick], *apierror.APIError) {
			return svc.(PickSvc).ListPicks
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePick,
			Fields:     []string{"sales_order", "customer", "departments"},
		}),
	})
}
