package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to update a transaction.
type UpdateTransactionRequest struct {
	// Transaction ID.
	TransactionID string `path:"id" validate:"required"`
	// Transaction number.
	Number *string `json:"number" nullable:"false" validate:"omitempty,max=255"`
	// Note.
	Note *string `json:"note" nullable:"false"`
	// Amount as a decimal string.
	Amount *string `json:"amount" nullable:"false"`
	// Transaction method code.
	TransactionMethodCode *string `json:"method" nullable:"false" validate:"omitempty,max=255"`
	// Adjustment type code.
	AdjustmentTypeCode *string `json:"adjustment_type" nullable:"false" validate:"omitempty,max=255"`
	// Responsible user ID.
	ResponsibleUserID *string `json:"responsible_user_id" nullable:"false" validate:"omitempty"`
	// Set to true to clear the responsible user.
	ClearResponsibleUser bool `json:"clear_responsible_user"`
	// Set to true to clear the transaction method.
	ClearTransactionMethod bool `json:"clear_transaction_method"`
	// Set to true to clear the adjustment type.
	ClearAdjustmentType bool `json:"clear_adjustment_type"`
	// Allocation status.
	IsFullyAllocated *bool `json:"is_fully_allocated" nullable:"false"`
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
		ServiceHandler: func(svc any) func(ctx context.Context, req *UpdateTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
			return svc.(TransactionSvc).UpdateTransaction
		},
	})
}
