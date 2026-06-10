package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	"github.com/augno/api/shared/constants"
	apierror "github.com/augno/api/shared/errors"
	"github.com/augno/api/shared/field"
)

// Request to update a transaction.
type UpdateTransactionRequest struct {
	// Transaction ID.
	TransactionID string `path:"id" validate:"required"`
	// Transaction number.
	Number field.Optional[string] `json:"number,omitzero" validate:"omitempty,max=255"`
	// Note.
	Note field.Optional[string] `json:"note,omitzero"`
	// Amount as a decimal string.
	Amount field.Optional[string] `json:"amount,omitzero"`
	// Transaction method code.
	TransactionMethodCode field.Optional[string] `json:"method,omitzero" validate:"omitempty,max=255"`
	// Adjustment type code.
	AdjustmentTypeCode field.Optional[string] `json:"adjustment_type,omitzero" validate:"omitempty,max=255"`
	// Responsible user ID.
	ResponsibleUserID field.Optional[string] `json:"responsible_user_id,omitzero" validate:"omitempty"`
	// Set to true to clear the responsible user.
	ClearResponsibleUser bool `json:"clear_responsible_user"`
	// Set to true to clear the transaction method.
	ClearTransactionMethod bool `json:"clear_transaction_method"`
	// Set to true to clear the adjustment type.
	ClearAdjustmentType bool `json:"clear_adjustment_type"`
	// Allocation status.
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
