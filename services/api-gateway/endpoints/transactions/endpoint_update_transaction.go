package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	"github.com/open-mrp/api/shared/constants"
	apierror "github.com/open-mrp/api/shared/errors"
	"github.com/open-mrp/api/shared/field"
)

// Request to update a transaction.
type UpdateTransactionRequest struct {
	// Transaction ID.
	TransactionID string `path:"id" validate:"required"`
	// New transaction number.
	//
	// Must be unique within the account; the request fails with a conflict error if another transaction already uses it.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Free-form note attached to the transaction.
	Note field.Optional[string] `json:"note,omitzero"`
	// New transaction amount as a decimal string, in US dollars.
	Amount field.Optional[string] `json:"amount,omitzero"`
	// How the money moved.
	TransactionMethodCode field.Optional[constants.TransactionMethod] `json:"method,omitzero" validate:"omitempty"`
	// The kind of correction this transaction represents (see List Adjustment Types for available values).
	AdjustmentTypeCode field.Optional[string] `json:"adjustment_type,omitzero" validate:"omitempty,max=255"`
	// ID of the account user responsible for the transaction.
	//
	// A user ID is also accepted; the value is resolved to an account user in the current account.
	ResponsibleUserID field.Optional[string] `json:"responsible_user_id,omitzero" validate:"omitempty"`
	// Set to true to clear the responsible user.
	//
	// Takes precedence over `responsible_user_id` if both are provided.
	ClearResponsibleUser bool `json:"clear_responsible_user"`
	// Set to true to clear the transaction method.
	//
	// Takes precedence over `method` if both are provided.
	ClearTransactionMethod bool `json:"clear_transaction_method"`
	// Set to true to clear the adjustment type.
	//
	// Takes precedence over `adjustment_type` if both are provided.
	ClearAdjustmentType bool `json:"clear_adjustment_type"`
	// Whether the full transaction amount has been applied to invoices.
	//
	// Set this to correct the flag by hand: editing or deleting individual allocations never recomputes it. While it is `false`, the transaction is returned by List Open Credits.
	IsFullyAllocated field.Optional[bool] `json:"is_fully_allocated,omitzero"`
}

var sampleUpdateTransactionNote = "Updated payment note"
var sampleUpdateTransactionAmount = "750.00"
var sampleUpdateTransactionMethodCode = constants.TransactionMethodACH
var sampleUpdateTransactionRequest = &UpdateTransactionRequest{
	Note:                  field.Some(sampleUpdateTransactionNote),
	Amount:                field.Some(sampleUpdateTransactionAmount),
	TransactionMethodCode: field.Some(sampleUpdateTransactionMethodCode),
}

func (*UpdateTransactionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateTransactionRequest)
}

// Updates a transaction, changing only the fields present in the request body.
//
// Changing the amount does not re-apply the transaction to invoices: existing allocations keep their amounts, and neither the transaction's `is_fully_allocated` flag nor the paid-in-full status of any settled invoice is recomputed.
type UpdateTransactionEndpoint struct{}

func (e *UpdateTransactionEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateTransactionRequest, *apiresource.TransactionDetail] {
	return (&apiendpoint.APIEndpoint[*UpdateTransactionRequest, *apiresource.TransactionDetail]{
		Title:             "Update Transaction",
		Method:            http.MethodPatch,
		ContentType:       "application/json",
		Route:             "/v1/finance/transactions/{id}",
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainTransactions, Action: types.ActionUpdate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
			return svc.(TransactionSvc).UpdateTransaction
		},
		ObjectType: constants.ObjectTypeTransaction,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTransaction,
			Fields:     []string{"allocations", "allocations.amount", "allocations.amount.unit", "allocations.transaction", "allocations.transaction.amount", "allocations.transaction.amount.unit", "customer", "responsible_user", "responsible_user.user"},
		}),
	})
}
