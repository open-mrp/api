package settlementep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/services/auth-service/pkg/types"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// A single allocation applying part of a transaction's amount to an invoice.
type CreateSettlementAllocationRequest struct {
	// ID of the transaction (payment, rebate, adjustment, or credit memo) to allocate from.
	TransactionID string `json:"transaction_id" validate:"required"`
	// ID of the invoice the amount is applied to.
	InvoiceID string `json:"invoice_id" validate:"required"`
	// The part of the transaction's amount to apply to this invoice, as a decimal string in US dollars.
	//
	// This is not checked against the transaction's unallocated balance or the invoice's outstanding total; applying more than an invoice owes leaves that invoice `overpaid`.
	Amount string `json:"amount" validate:"required"`
	// Free-form note about this allocation.
	Note field.Optional[string] `json:"note,omitzero"`
}

// Request to create a settlement.
type CreateSettlementRequest struct {
	// ID of the user responsible for this settlement.
	//
	// Accepts either an account user ID or a user ID; the value is resolved to an account user in the current account.
	ResponsibleUserID string `json:"responsible_user_id" validate:"required"`
	// Allocations to record in this settlement.
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

// Creates a settlement that applies transaction amounts to invoices.
//
// The settlement number is generated automatically from a per-account sequence.
//
// Once the settlement is recorded, every transaction it drew from is marked fully allocated even if only part of its amount was applied, which drops it out of List Open Credits. Each invoice it touched has its paid-in-full and overpaid flags — and therefore its `payment_status` — recomputed from every allocation recorded against that invoice, including allocations made by other settlements.
type CreateSettlementEndpoint struct{}

func (e *CreateSettlementEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateSettlementRequest, *apiresource.Settlement] {
	return (&apiendpoint.APIEndpoint[*CreateSettlementRequest, *apiresource.Settlement]{
		Title:               "Create Settlement",
		Method:              http.MethodPost,
		ContentType:         "application/json",
		Route:               "/v1/finance/settlements",
		SuccessStatusCode:   http.StatusCreated,
		Public:              false,
		Preview:             true,
		RequiredPermissions: []types.Permission{{Domain: types.PermissionDomainSettlements, Action: types.ActionCreate}},
		ObjectType:          constants.ObjectTypeSettlement,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateSettlementRequest) (*apiresource.Settlement, *apierror.APIError) {
			return svc.(SettlementSvc).CreateSettlement
		},
		LocationFunc: func(resp *apiresource.Settlement) string {
			return "/v1/finance/settlements/" + resp.ID
		},
	})
}
