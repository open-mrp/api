package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a production run from a sales order.
type CreateProductionRunRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
}

// Creates a production run from a sales order.
//
// Creates a batch for each of the order's item-backed lines, reserves the material inventory required to produce them, and links the run to the order. An order can have at most one production run.
type CreateProductionRunEndpoint struct{}

func (e *CreateProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductionRunRequest, *apiresource.ProductionRun] {
	return (&apiendpoint.APIEndpoint[*CreateProductionRunRequest, *apiresource.ProductionRun]{
		Title:               "Create Production Run from Sales Order",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/sales/sales-orders/{id}/actions/create-production-run",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		Preview:             true,
		ObjectType:          constants.ObjectTypeProductionRun,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainProductionRuns, Action: types.ActionCreate}},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductionRunRequest) (*apiresource.ProductionRun, *apierror.APIError) {
			return svc.(SalesOrderSvc).CreateSalesOrderProductionRun
		},
		LocationFunc: func(resp *apiresource.ProductionRun) string {
			return "/v1/operations/production-runs/" + resp.ID
		},
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeProductionRun,
			Fields:     []string{"responsible_user", "responsible_user.user"},
		}),
	})
}
