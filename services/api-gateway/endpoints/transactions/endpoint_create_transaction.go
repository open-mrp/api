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

// Request to create a transaction.
type CreateTransactionRequest struct {
	// ID of the customer the transaction is recorded against.
	CustomerID string `json:"customer_id" validate:"required"`
	// Transaction type code.
	//
	// - `payment`: money received from the customer.
	// - `credit_memo`: a credit issued to the customer.
	// - `adjustment`: a manual correction (also provide `adjustment_type`).
	// - `rebate`: a rebate granted to the customer.
	TransactionTypeCode string `json:"type" validate:"required,max=255"`
	// Transaction amount as a decimal string, in US dollars.
	Amount string `json:"amount" validate:"required"`
	// Payment method code: one of `cash`, `check`, `credit_card`, `gift_card`, or `ach`.
	//
	// Typically provided for payment transactions.
	TransactionMethodCode field.Optional[string] `json:"method,omitzero" validate:"omitempty,max=255"`
	// Adjustment type code (see List Adjustment Types for available values).
	//
	// Typically provided when `type` is `adjustment`.
	AdjustmentTypeCode field.Optional[string] `json:"adjustment_type,omitzero" validate:"omitempty,max=255"`
	// ID of the account user responsible for the transaction.
	//
	// When omitted, the account user making the request is recorded as responsible.
	ResponsibleUserID field.Optional[string] `json:"responsible_user_id,omitzero" validate:"omitempty"`
	// Free-form note attached to the transaction.
	Note field.Optional[string] `json:"note,omitzero"`
}

var sampleCreateTransactionMethodCode = "check"
var sampleCreateTransactionNote = "Q1 invoice payment"
var sampleCreateTransactionRequest = &CreateTransactionRequest{
	CustomerID:            apiresource.SampleCustomerID,
	TransactionTypeCode:   "payment",
	Amount:                "500.00",
	TransactionMethodCode: field.Some(sampleCreateTransactionMethodCode),
	Note:                  field.Some(sampleCreateTransactionNote),
}

func (*CreateTransactionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateTransactionRequest)
}

// Creates a transaction with an automatically generated transaction number.
type CreateTransactionEndpoint struct{}

func (e *CreateTransactionEndpoint) Materialize() *apiendpoint.APIEndpoint[*CreateTransactionRequest, *apiresource.TransactionDetail] {
	return (&apiendpoint.APIEndpoint[*CreateTransactionRequest, *apiresource.TransactionDetail]{
		Title:             "Create Transaction",
		Method:            http.MethodPost,
		ContentType:       "application/json",
		Route:             "/v1/finance/transactions",
		SuccessStatusCode: http.StatusCreated,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainTransactions, Action: types.ActionCreate},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *CreateTransactionRequest) (*apiresource.TransactionDetail, *apierror.APIError) {
			return svc.(TransactionSvc).CreateTransaction
		},
		LocationFunc: func(resp *apiresource.TransactionDetail) string {
			return "/v1/finance/transactions/" + resp.ID
		},
		ObjectType: constants.ObjectTypeTransaction,
		IncludeConfig: apiendpoint.IncludesFor(apiendpoint.IncludesParams{
			ObjectType: constants.ObjectTypeTransaction,
			Fields:     []string{"allocations", "customer", "responsible_user", "responsible_user.user"},
		}),
	})
}
