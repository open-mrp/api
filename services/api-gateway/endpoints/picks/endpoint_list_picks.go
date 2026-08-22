package pickep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to list picks.
type ListPicksRequest struct {
	apiresource.PaginationRequest
	// Filter by pick status.
	//
	// Pass `open` for picks that have not been finished, or `closed` for picks that have.
	Status *string `query:"status"`
	// Filter by customer IDs.
	CustomerIDs []string `query:"customer_ids"`
	// Filter by product line IDs.
	//
	// Matches picks that contain at least one line for a product in any of the given product lines.
	ProductLineIDs []string `query:"product_line_ids"`
	// Filter by customer type group IDs.
	//
	// Matches picks whose customer's type group — the account group returned in the customer's `type` field — is one of the given groups.
	CustomerGroupIDs []string `query:"customer_group_ids"`
	// Filter by department IDs.
	//
	// Matches picks assigned to any of the given departments.
	DepartmentIDs []string `query:"department_ids"`
	// Only return picks created on or after this date (`YYYY-MM-DD`).
	StartDate *string `query:"starts_at"`
	// Only return picks created before this date (`YYYY-MM-DD`).
	EndDate *string `query:"ends_at"`
	// Orders the results: `ship_by_date` puts the soonest delivery commitment first, with picks whose order has no ship-by date last; `created_at` puts the newest pick first.
	Sort constants.PickSort `query:"sort" default:"ship_by_date"`
}

// Returns a paginated list of picks, soonest ship-by date first.
//
// The `q` search term matches the pick number, the sales order number, the customer PO number, and the customer's name or number.
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
			// Same resource as detail, so the same include set — a row just asks for less.
			Fields: []string{
				"customer", "related.sales_order", "related.shipments",

				"lines",
				"lines.item",
				"lines.sales_order_line",
				"lines.sales_order_line.product",
				"lines.quantity",
				"lines.quantity.unit",
				"lines.ordered_quantity",
				"lines.ordered_quantity.unit",
			},
		}),
	})
}
