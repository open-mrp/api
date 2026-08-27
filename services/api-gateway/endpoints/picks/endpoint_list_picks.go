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
	// Restricts results to picks in this state.
	//
	// - `open`: picks that have not been finished.
	// - `closed`: picks that have been finished.
	Status *constants.PickStatus `query:"status"`
	// Restricts results to picks raised for any of these customers.
	CustomerIDs []string `query:"customer_ids"`
	// Restricts results to picks with at least one line whose product belongs to any of these product lines.
	ProductLineIDs []string `query:"product_line_ids"`
	// Restricts results to picks whose customer belongs to any of these account groups, matching the `type` on the customer.
	CustomerGroupIDs []string `query:"customer_group_ids"`
	// Earliest pick creation date to include, in `YYYY-MM-DD` format.
	StartDate *string `query:"starts_at" validate:"omitempty,date_filter"`
	// Latest pick creation date to include, in `YYYY-MM-DD` format. Inclusive of the date itself.
	EndDate *string `query:"ends_at" validate:"omitempty,date_filter"`
	// Orders the results: `ship_by_date` puts the soonest delivery commitment first, with picks whose order has no ship-by date last; `created_at` puts the newest pick first.
	Sort *constants.PickSort `query:"sort" default:"ship_by_date"`
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
		Public:            true,
		Preview:           true,
		AgentTool:         true,
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
				"customer", "created_by", "freight", "related.sales_order", "related.shipments",

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
