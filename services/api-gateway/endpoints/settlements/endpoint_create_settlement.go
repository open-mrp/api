package settlementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateSettlementAllocationRequest is an allocation in a create settlement request.
type CreateSettlementAllocationRequest struct {
	// Transaction ID.
	TransactionID string `json:"transaction_id" validate:"required"`
	// Invoice ID.
	InvoiceID string `json:"invoice_id" validate:"required"`
	// Amount to allocate as a decimal string.
	Amount string `json:"amount" validate:"required"`
	// Note about this allocation.
	Note *string `json:"note"`
}

// CreateSettlementRequest is the request to create a settlement.
type CreateSettlementRequest struct {
	// Responsible user ID.
	ResponsibleUserID string `json:"responsible_user_id" validate:"required"`
	// Allocations for this settlement.
	Allocations []CreateSettlementAllocationRequest `json:"allocations" validate:"required,min=1"`
}

var sampleCreateSettlementRequest = &CreateSettlementRequest{
	ResponsibleUserID: apiresource.SampleUserID,
	Allocations: []CreateSettlementAllocationRequest{
		{
			TransactionID: apiresource.SampleTransactionDetailID,
			InvoiceID:     apiresource.SampleInvoiceID,
			Amount:        "150.00",
		},
	},
}

func (*CreateSettlementRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateSettlementRequest)
}

type CreateSettlementEndpoint struct{}

func (e *CreateSettlementEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSettlementRequest, *apiresource.Settlement] {
	return &apiendpoint.APIEndpoint[*CreateSettlementRequest, *apiresource.Settlement]{
		Title:             "Create Settlement",
		Description:       "Creates a settlement with transaction allocations. A settlement number is automatically generated.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/finance/settlements",
		Request:           &CreateSettlementRequest{},
		Response:          &apiresource.Settlement{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
			return svc.(SettlementSvc).CreateSettlement
		},
		LocationFunc: func(resp *apiresource.Settlement) string {
			return "/v1/finance/settlements/" + resp.ID
		},
	}
}
