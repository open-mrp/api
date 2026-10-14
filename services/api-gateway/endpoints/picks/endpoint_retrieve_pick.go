package pickep

import (
	"context"
	"net/http"

	"github.com/open-mrp/api/services/api-gateway/pkg/authz"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/apikit/apierror"
	apiendpoint "github.com/open-mrp/apikit/endpoint"
)

// Request to retrieve a pick.
type RetrievePickRequest struct {
	// Pick ID.
	PickID string `path:"id" validate:"required"`
}

// Returns a pick by ID.
type RetrievePickEndpoint struct{}

func (e *RetrievePickEndpoint) Materialize() *apiendpoint.APIEndpoint[*RetrievePickRequest, *apiresource.Pick] {
	return (&apiendpoint.APIEndpoint[*RetrievePickRequest, *apiresource.Pick]{
		Title:             "Retrieve Pick",
		Method:            http.MethodGet,
		ContentType:       "application/json",
		Route:             "/v1/operations/picks/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            true,
		Preview:           true,
		AgentTool:         true,
		ObjectType:        constants.ObjectTypePick,
		Auth: authz.Policy{Permissions: []types.Permission{
			{Domain: types.PermissionDomainPicks, Action: types.ActionRead},
		}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *RetrievePickRequest) (*apiresource.Pick, *apierror.APIError) {
			return svc.(PickSvc).GetPick
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypePick,
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
