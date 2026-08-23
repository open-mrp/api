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
	TransactionTypeCode constants.TransactionType `json:"type" validate:"required"`
	// Transaction amount as a decimal string, in US dollars.
	Amount string `json:"amount" validate:"required"`
	// How the money moved.
	//
	// Typically provided for payment transactions.
	TransactionMethodCode field.Optional[constants.TransactionMethod] `json:"method,omitzero" validate:"omitempty"`
	// The kind of correction this transaction represents (see List Adjustment Types for available values).
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

var sampleCreateTransactionMethodCode = constants.TransactionMethodCheck
var sampleCreateTransactionNote = "Q1 invoice payment"
var sampleCreateTransactionRequest = &CreateTransactionRequest{
	CustomerID:            apiresource.SampleCustomerID,
	TransactionTypeCode:   constants.TransactionTypePayment,
	Amount:                "500.00",
	TransactionMethodCode: field.Some(sampleCreateTransactionMethodCode),
	Note:                  field.Some(sampleCreateTransactionNote),
}

func (*CreateTransactionRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleCreateTransactionRequest)
}

// Records a financial transaction against a customer, such as a payment received, a credit memo, an adjustment, or a rebate.
//
// The transaction number is assigned automatically from the account's transaction sequence. The new transaction starts out unapplied, so it shows up as an open credit until it is applied to invoices by recording a settlement.
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
