package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// CreateTransactionRequest is the request to create a new transaction.
type CreateTransactionRequest struct {
	// The ID of the customer for this transaction.
	CustomerID string `json:"customer_id" validate:"required"`
	// The transaction type code.
	TransactionTypeCode string `json:"transaction_type_code" validate:"required"`
	// The transaction amount as a decimal string.
	Amount string `json:"amount" validate:"required"`
	// The transaction method code.
	TransactionMethodCode *string `json:"transaction_method_code"`
	// The adjustment type code, if applicable.
	AdjustmentTypeCode *string `json:"adjustment_type_code"`
	// The ID of the user responsible for this transaction.
	ResponsibleUserID *string `json:"responsible_user_id"`
	// A note about this transaction.
	Note *string `json:"note"`
}

var sampleCreateTransactionMethodCode = "check"
var sampleCreateTransactionNote = "Q1 invoice payment"
var sampleCreateTransactionRequest = &CreateTransactionRequest{
	CustomerID:            apiresource.SampleCustomerID,
	TransactionTypeCode:   "payment",
	Amount:                "500.00",
	TransactionMethodCode: &sampleCreateTransactionMethodCode,
	Note:                  &sampleCreateTransactionNote,
}

func (*CreateTransactionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateTransactionRequest)
}

type CreateTransactionEndpoint struct{}

func (e *CreateTransactionEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateTransactionRequest, *apiresource.TransactionDetail] {
	return &apiendpoint.APIEndpoint[*CreateTransactionRequest, *apiresource.TransactionDetail]{
		Title:             "Create Transaction",
		Description:       "Creates a new transaction with an automatically generated transaction number.",
		Method:            http.MethodPost,
		Route:             "/v1/finance/transactions",
		Request:           &CreateTransactionRequest{},
		Response:          &apiresource.TransactionDetail{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
			return svc.(TransactionSvc).CreateTransaction
		},
	}
}
