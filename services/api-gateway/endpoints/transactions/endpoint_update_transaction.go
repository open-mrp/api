package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// UpdateTransactionRequest is the request to update a transaction.
type UpdateTransactionRequest struct {
	// The ID of the transaction to update.
	TransactionID string `path:"id" validate:"required"`
	// The new transaction number.
	Number *string `json:"number"`
	// The new note for this transaction.
	Note *string `json:"note"`
	// The new amount as a decimal string.
	Amount *string `json:"amount"`
	// The new transaction method code.
	TransactionMethodCode *string `json:"transaction_method_code"`
	// The new adjustment type code.
	AdjustmentTypeCode *string `json:"adjustment_type_code"`
	// The new responsible user ID.
	ResponsibleUserID *string `json:"responsible_user_id"`
	// Set to true to clear the responsible user.
	ClearResponsibleUser bool `json:"clear_responsible_user"`
	// Set to true to clear the transaction method.
	ClearTransactionMethod bool `json:"clear_transaction_method"`
	// Set to true to clear the adjustment type.
	ClearAdjustmentType bool `json:"clear_adjustment_type"`
	// The allocation status of the transaction.
	IsFullyAllocated *bool `json:"is_fully_allocated"`
}

var sampleUpdateTransactionNote = "Updated payment note"
var sampleUpdateTransactionAmount = "750.00"
var sampleUpdateTransactionMethodCode = "ach"
var sampleUpdateTransactionRequest = &UpdateTransactionRequest{
	Note:                  &sampleUpdateTransactionNote,
	Amount:                &sampleUpdateTransactionAmount,
	TransactionMethodCode: &sampleUpdateTransactionMethodCode,
}

func (*UpdateTransactionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleUpdateTransactionRequest)
}

type UpdateTransactionEndpoint struct{}

func (e *UpdateTransactionEndpoint) Materialize() *apiendpoint.APIEndpoint[*UpdateTransactionRequest, *apiresource.TransactionDetail] {
	return &apiendpoint.APIEndpoint[*UpdateTransactionRequest, *apiresource.TransactionDetail]{
		Title:             "Update Transaction",
		Description:       "Partially updates a transaction.",
		Method:            http.MethodPatch,
		Route:             "/v1/finance/transactions/{id}",
		Request:           &UpdateTransactionRequest{},
		Response:          &apiresource.TransactionDetail{},
		SuccessStatusCode: http.StatusOK,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
			return svc.(TransactionSvc).UpdateTransaction
		},
	}
}
