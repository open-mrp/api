package receivableep

import (
	"context"
	"net/http"

	apiendpoint "github.com/open-mrp/api/services/api-gateway/pkg/endpoint"
	apiexample "github.com/open-mrp/api/services/api-gateway/pkg/example"
	apiresource "github.com/open-mrp/api/services/api-gateway/pkg/resource"
	"github.com/open-mrp/api/services/auth-service/pkg/types"
	apierror "github.com/open-mrp/api/shared/errors"
)

// Request to email a statement of account for a specific customer.
type EmailReceivablesForCustomerRequest struct {
	// ID of the customer account the statement is prepared for.
	AccountID string `json:"-" path:"account_id" validate:"required"`
	// Email addresses to send the statement of account to.
	//
	// The statement goes only to these addresses; the customer's own notification contacts are not added.
	RecipientEmails []string `json:"recipient_emails" validate:"required,min=1"`
}

var sampleEmailReceivablesForCustomerRequest = &EmailReceivablesForCustomerRequest{
	RecipientEmails: []string{apiresource.SampleUserEmail},
}

func (*EmailReceivablesForCustomerRequest) SchemaExample() any {
	return apiexample.ValidateAndMarshalToMap(sampleEmailReceivablesForCustomerRequest)
}

// Emails a statement of account for a specific customer to the provided recipients.
//
// The email carries an Excel attachment listing the customer's outstanding receivables and its open credits, which are transactions such as payments and credit memos that still have an unapplied balance. The statement always reflects current balances; there is no cutoff date. Delivery is asynchronous: the endpoint returns `202 Accepted` once the email is queued.
type EmailReceivablesForCustomerEndpoint struct{}

func (e *EmailReceivablesForCustomerEndpoint) Materialize() *apiendpoint.APIEndpoint[*EmailReceivablesForCustomerRequest, *apiresource.EmptyResource] {
	return (&apiendpoint.APIEndpoint[*EmailReceivablesForCustomerRequest, *apiresource.EmptyResource]{
		Title:             "Email Receivables for Customer",
		Method:            http.MethodPost,
		Route:             "/v1/finance/accounts/{account_id}/actions/email-receivables",
		ContentType:       "application/json",
		SuccessStatusCode: http.StatusAccepted,
		Public:            false,
		Preview:           true,
		RequiredPermissions: []types.Permission{
			{Domain: types.PermissionDomainCustomers, Action: types.ActionRead},
		},
		ServiceHandler: func(svc any) func(ctx context.Context, req *EmailReceivablesForCustomerRequest) (*apiresource.EmptyResource, *apierror.APIError) {
			return svc.(ReceivableSvc).EmailReceivablesForCustomer
		},
	})
}
