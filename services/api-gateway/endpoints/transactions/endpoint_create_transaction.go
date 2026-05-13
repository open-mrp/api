package transactionep

import (
	"context"
	"net/http"

	apiendpoint "github.com/augno/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/augno/api/services/api-gateway/pkg/example"
	apiresource "github.com/augno/api/services/api-gateway/pkg/resource"
	apierror "github.com/augno/api/shared/errors"
)

// Request to create a transaction.
type CreateTransactionRequest struct {
	// Customer ID.
	CustomerID string `json:"customer_id" validate:"required"`
	// Transaction type code.
	TransactionTypeCode string `json:"type" validate:"required,max=255"`
	// Transaction amount as a decimal string.
	Amount string `json:"amount" validate:"required"`
	// Transaction method code.
	TransactionMethodCode *string `json:"method" validate:"omitempty,max=255"`
	// Adjustment type code.
	AdjustmentTypeCode *string `json:"adjustment_type" validate:"omitempty,max=255"`
	// Responsible user ID.
	ResponsibleUserID *string `json:"responsible_user_id" validate:"omitempty"`
	// Note.
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
		Description:       "Creates a transaction with an automatically generated transaction number.",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/finance/transactions",
		Request:           &CreateTransactionRequest{},
		Response:          &apiresource.TransactionDetail{},
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
			return svc.(TransactionSvc).CreateTransaction
		},
		LocationFunc: func(resp *apiresource.TransactionDetail) string {
			return "/v1/finance/transactions/" + resp.ID
		},
	}
}
