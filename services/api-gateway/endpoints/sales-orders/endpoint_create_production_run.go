package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateProductionRunRequest is the request to create a production run from a sales order.
type CreateProductionRunRequest struct {
	// The ID of the sales order.
	SalesOrderID string `path:"id" validate:"required"`
}

// CreateProductionRunResponseRef is a lightweight reference to the created production run.
type CreateProductionRunResponseRef struct {
	// The unique identifier of the created production run.
	ID string `json:"id" validate:"required,max=191"`
}

// CreateProductionRunResponse represents the result of creating a production run.
type CreateProductionRunResponse struct {
	// The created production run.
	ProductionRun CreateProductionRunResponseRef `json:"production_run" validate:"required"`
}

func (*CreateProductionRunResponse) SchemaExample() any {
	return map[string]any{"production_run": map[string]any{"id": apiresource.SampleProductionRunID}}
}

type CreateProductionRunEndpoint struct{}

func (e *CreateProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductionRunRequest, *CreateProductionRunResponse] {
	return &apiendpoint.APIEndpoint[*CreateProductionRunRequest, *CreateProductionRunResponse]{
		Title:             "Create Production Run from Sales Order",
		Description:       "Creates a production run from a sales order.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/actions/create-production-run",
		Request:           &CreateProductionRunRequest{},
		Response:          &CreateProductionRunResponse{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductionRunRequest) (*CreateProductionRunResponse, *apierror.APIError) {
			return svc.(SalesOrderSvc).CreateSalesOrderProductionRun
		},
	}
}
