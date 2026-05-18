package salesorderep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a production run from a sales order.
type CreateProductionRunRequest struct {
	// Sales order ID.
	SalesOrderID string `path:"id" validate:"required"`
}

// Lightweight reference to a production run.
type CreateProductionRunResponseRef struct {
	// Production run ID.
	ID string `json:"id" validate:"required"`
	// Resource type identifier.
	Object constants.ObjectType `json:"object" validate:"required,enum=production_run"`
}

// Result of creating a production run.
type CreateProductionRunResponse struct {
	// Created production run.
	ProductionRun CreateProductionRunResponseRef `json:"production_run" validate:"required"`
}

func (*CreateProductionRunResponse) SchemaExample() any {
	return map[string]any{"production_run": map[string]any{"id": apiresource.SampleProductionRunID, "object": string(constants.ObjectTypeProductionRun)}}
}

// Creates a production run from a sales order.
type CreateProductionRunEndpoint struct{}

func (e *CreateProductionRunEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateProductionRunRequest, *CreateProductionRunResponse] {
	return (&apiendpoint.APIEndpoint[*CreateProductionRunRequest, *CreateProductionRunResponse]{
		Title:             "Create Production Run from Sales Order",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/sales/sales-orders/{id}/actions/create-production-run",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateProductionRunRequest) (*CreateProductionRunResponse, *apierror.APIError) {
			return svc.(SalesOrderSvc).CreateSalesOrderProductionRun
		},
	})
}
