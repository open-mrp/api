package settlementep

import (
	"context"
	"net/http"
	"time"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// A single allocation applying part of a transaction's amount to an invoice.
type CreateSettlementAllocationRequest struct {
	// ID of the transaction (payment, rebate, adjustment, or credit memo) to allocate from. Omit it and set `transaction_key` to allocate from one of the settlement's `new_transactions`.
	TransactionID field.Optional[string] `json:"transaction_id,omitzero"`
	// Key of the entry in `new_transactions` to allocate from, instead of an existing transaction.
	TransactionKey field.Optional[string] `json:"transaction_key,omitzero"`
	// ID of the invoice the amount is applied to.
	InvoiceID string `json:"invoice_id" validate:"required"`
	// The part of the transaction's amount to apply to this invoice, as a decimal string in US dollars.
	//
	// This is not checked against the transaction's unallocated balance or the invoice's outstanding total; applying more than an invoice owes leaves that invoice `overpaid`.
	Amount string `json:"amount" validate:"required"`
	// Free-form note about this allocation.
	Note field.Optional[string] `json:"note,omitzero"`
	// When the amount was applied, reported as the allocation's `created_at`; defaults to now.
	AppliedAt field.Optional[time.Time] `json:"applied_at,omitzero"`
}

// A transaction recorded together with the settlement that applies it, such as an adjustment or credit entered while settling.
//
// Its amount is the sum of the allocations naming its key. It is dated, and its funds counted as received, at the first of those allocations.
type NewSettlementTransactionRequest struct {
	// Names the transaction within this request; allocations draw on it through `transaction_key`.
	Key string `json:"key" validate:"required"`
	// Type of the transaction.
	TransactionTypeCode constants.TransactionType `json:"type" validate:"required"`
	// How the money moved, for a payment.
	TransactionMethodCode field.Optional[constants.TransactionMethod] `json:"method,omitzero" validate:"omitempty"`
	// Kind of adjustment, for an adjustment.
	AdjustmentTypeCode field.Optional[string] `json:"adjustment_type,omitzero" validate:"omitempty,max=255"`
	// The customer the transaction belongs to.
	CustomerID string `json:"customer_id" validate:"required"`
}

// Request to create a settlement.
type CreateSettlementRequest struct {
	// ID of the user responsible for this settlement.
	//
	// Accepts either an account user ID or a user ID; the value is resolved to an account user in the current account.
	ResponsibleUserID string `json:"responsible_user_id" validate:"required"`
	// Allocations to record in this settlement.
	Allocations []CreateSettlementAllocationRequest `json:"allocations" validate:"required,min=1"`
	// Transactions to record with the settlement.
	NewTransactions []NewSettlementTransactionRequest `json:"new_transactions,omitzero"`
}

var sampleCreateSettlementRequest = &CreateSettlementRequest{
	ResponsibleUserID: apiresource.SampleUserID,
	Allocations: []CreateSettlementAllocationRequest{
		{
			TransactionID: field.Some(apiresource.SampleTransactionDetailID),
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
		AgentTool:           true,
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
