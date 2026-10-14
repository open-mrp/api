package pickep

import (
	"context"
	"net/http"

	"github.com/open-mrp/apikit/object"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to list picks.
type ListPicksRequest struct {
	object.ListRequest
	// Free-text search term used to filter results.
	//
	// Which fields are matched against the term varies by endpoint.
	Query *string `query:"q" validate:"omitempty,max=500"`
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
	// How `q` is matched against the pick number and the customer PO number. Defaults to `prefix`.
	//
	// - `prefix`: `q` matches the start of the pick number and of the customer PO number.
	// - `contains`: `q` matches anywhere in the pick number and the customer PO number. Slower on accounts with many picks.
	//
	// A `q` of one or two characters matches the start of the pick number only, whichever mode is sent.
	QMatch *constants.PickSearchMatch `query:"q_match"`
}

// Returns a paginated list of picks, soonest ship-by date first.
//
// The `q` search term matches the start of the pick number (which is the order number) and of the customer PO number, ignoring case; send `q_match=contains` to match anywhere in them. A term of three or more characters also matches anywhere in the customer's name and customer number. To narrow by customer, use `customer_ids` rather than searching for a customer name.
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
		Auth:              authz.Policy{Permissions: []types.Permission{{Domain: types.PermissionDomainPicks, Action: types.ActionRead}}, Counterparty: authz.Counterparties(types.ActionRead)},
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
