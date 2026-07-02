package transactionep

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
	// Payment method code: one of `cash`, `check`, `credit_card`, `gift_card`, or `ach`.
	TransactionMethodCode field.Optional[string] `json:"method,omitzero" validate:"omitempty,max=255"`
	// Adjustment type code (see List Adjustment Types for available values).
	AdjustmentTypeCode field.Optional[string] `json:"adjustment_type,omitzero" validate:"omitempty,max=255"`
	// ID of the account user responsible for the transaction.
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
	// Whether the full transaction amount has been allocated against invoices.
	//
	// This flag is set explicitly here; it is not recomputed automatically when allocations change.
	IsFullyAllocated field.Optional[bool] `json:"is_fully_allocated,omitzero"`
}

var sampleUpdateTransactionNote = "Updated payment note"
var sampleUpdateTransactionAmount = "750.00"
var sampleUpdateTransactionMethodCode = "ach"
var sampleUpdateTransactionRequest = &UpdateTransactionRequest{
	Note:                  field.Some(sampleUpdateTransactionNote),
	Amount:                field.Some(sampleUpdateTransactionAmount),
	TransactionMethodCode: field.Some(sampleUpdateTransactionMethodCode),
}

func (*UpdateTransactionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateTransactionRequest)
}

// Partially updates a transaction.
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
			Fields:     []string{"allocations", "customer", "responsible_user", "responsible_user.user"},
		}),
	})
}
