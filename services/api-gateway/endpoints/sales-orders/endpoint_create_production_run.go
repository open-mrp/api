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
// Walks the production flow behind each item-backed line to work out what actually has to be made, then creates one batch for each item that is produced directly from raw materials, sized to cover every line that needs it. Reserves the material inventory those batches consume and links the run to the order. The caller becomes the run's responsible user. An order can have at most one production run, and a line whose item has no production flow contributes no batches.
type CreateProductionRunEndpoint struct{}

func (e *CreateProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductionRunRequest, *apiresource.ProductionRun] {
	return (&apiendpoint.APIEndpoint[*CreateProductionRunRequest, *apiresource.ProductionRun]{
		Title:               "Create Production Run from Sales Order",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/sales/sales-orders/{id}/actions/create-production-run",
		SuccessStatusCode:   http.StatusCreated,
		Public:              true,
		AgentTool:           true,
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
